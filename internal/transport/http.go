package transport

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/muz/xadventure/internal/config"
	"github.com/muz/xadventure/internal/domain"
	"github.com/muz/xadventure/internal/service"
)

func SetupRouter(engine *service.Engine, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Heartbeat("/health"))
	r.Use(middleware.RequestSize(cfg.RequestMaxBytes))
	r.Use(corsMiddleware)

	// Serve static files with cache-busting headers
	fs := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", cacheControlMiddleware(http.StripPrefix("/static/", fs)))

	// Serve index.html
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Vary", "*")
		http.ServeFile(w, r, "./web/templates/index.html")
	})

	r.Get("/kids", func(w http.ResponseWriter, r *http.Request) {
		if !cfg.FeatureEnabled("kids_mode") {
			http.Error(w, "Kids mode is not enabled", http.StatusNotFound)
			return
		}
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Vary", "*")
		http.ServeFile(w, r, "./web/templates/kids.html")
	})

	r.Post("/api/start", handleStart(engine, cfg))
	r.Post("/api/turn", handleTurn(engine, cfg))
	r.Get("/api/kids/image", handleKidsImage(cfg))
	r.Get("/api/session/{sessionID}", handleGetSession(engine))
	r.Get("/api/download/{sessionID}", handleDownload(engine))

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func cacheControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

func handleKidsImage(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.FeatureEnabled("kids_mode") {
			http.Error(w, "Kids mode is not enabled", http.StatusNotFound)
			return
		}
		prompt := strings.TrimSpace(r.URL.Query().Get("prompt"))
		if prompt == "" {
			http.Error(w, "prompt is required", http.StatusBadRequest)
			return
		}
		if !cfg.HasImageRouterProvider() {
			http.Error(w, "image generation is not configured", http.StatusServiceUnavailable)
			return
		}

		payload := map[string]interface{}{
			"model":         cfg.ImageRouterModel,
			"prompt":        prompt,
			"quality":       "low",
			"size":          "1024x1024",
			"output_format": "jpeg",
		}
		body, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, "failed to build image request", http.StatusInternalServerError)
			return
		}

		base := strings.TrimRight(cfg.ImageRouterBase, "/")
		slog.Info("kids_image_provider_selected",
			"provider_base", cfg.ImageRouterBase,
			"endpoint", base+"/images/generations",
			"model", cfg.ImageRouterModel,
		)
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, base+"/images/generations", bytes.NewReader(body))
		if err != nil {
			http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.ImageRouterKey)
		req.Header.Set("User-Agent", "xadventure/1.0")

		client := &http.Client{Timeout: time.Duration(cfg.LLMTimeoutSec) * time.Second}
		if cfg.LLMTimeoutSec <= 0 {
			client.Timeout = 120 * time.Second
		}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "image generation failed", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			http.Error(w, "image provider error: "+string(msg), http.StatusBadGateway)
			return
		}

		if err := writeImageResponse(w, r, client, resp, base); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
}

type imageGenerationResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
}

func writeImageResponse(w http.ResponseWriter, r *http.Request, client *http.Client, upstream *http.Response, providerBase string) error {
	contentType := upstream.Header.Get("Content-Type")
	if strings.HasPrefix(strings.ToLower(contentType), "image/") {
		imageBytes, err := io.ReadAll(upstream.Body)
		if err != nil {
			return errors.New("failed to read image bytes")
		}
		imageBytes, contentType = normalizeKidsStoryImage(imageBytes, contentType)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "private, max-age=86400")
		_, _ = w.Write(imageBytes)
		return nil
	}

	body, err := io.ReadAll(upstream.Body)
	if err != nil {
		return errors.New("failed to read image response")
	}

	var payload imageGenerationResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("kids_image_unexpected_response",
			"provider_base", providerBase,
			"content_type", contentType,
			"body_preview", string(body[:min(len(body), 200)]),
		)
		return errors.New("image provider returned an unexpected response format")
	}
	if len(payload.Data) == 0 {
		return errors.New("image provider returned no image data")
	}

	item := payload.Data[0]
	switch {
	case strings.TrimSpace(item.B64JSON) != "":
		imageBytes, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return errors.New("failed to decode image data")
		}
		imageBytes, contentType = normalizeKidsStoryImage(imageBytes, "image/jpeg")
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "private, max-age=86400")
		_, _ = w.Write(imageBytes)
		return nil
	case strings.TrimSpace(item.URL) != "":
		return proxyRemoteImage(w, r, client, providerBase, item.URL)
	default:
		return errors.New("image provider returned empty image data")
	}
}

func proxyRemoteImage(w http.ResponseWriter, r *http.Request, client *http.Client, providerBase, imageURL string) error {
	resolvedURL, err := resolveImageURL(providerBase, imageURL)
	if err != nil {
		return errors.New("image provider returned an invalid image URL")
	}
	slog.Info("kids_image_fetching_remote_asset", "provider_base", providerBase, "asset_url", resolvedURL)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, resolvedURL, nil)
	if err != nil {
		return errors.New("failed to build image asset request")
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("failed to fetch remote image asset")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("remote image asset request failed")
	}

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.New("failed to read remote image asset")
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		contentType = "image/jpeg"
	}
	imageBytes, contentType = normalizeKidsStoryImage(imageBytes, contentType)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(imageBytes)
	return nil
}

func resolveImageURL(providerBase, imageURL string) (string, error) {
	if strings.TrimSpace(imageURL) == "" {
		return "", errors.New("empty image URL")
	}
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() {
		return parsed.String(), nil
	}
	base, err := url.Parse(strings.TrimRight(providerBase, "/") + "/")
	if err != nil {
		return "", err
	}
	return base.ResolveReference(parsed).String(), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeKidsStoryImage(imageBytes []byte, contentType string) ([]byte, string) {
	img, format, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return imageBytes, contentType
	}

	rect, ok := detectLargestPanelRect(img)
	if !ok {
		return imageBytes, contentType
	}

	cropped := cropImage(img, rect)
	var out bytes.Buffer
	switch strings.ToLower(format) {
	case "png":
		if err := png.Encode(&out, cropped); err != nil {
			return imageBytes, contentType
		}
		slog.Info("kids_image_panel_crop_applied", "format", format, "width", rect.Dx(), "height", rect.Dy())
		return out.Bytes(), "image/png"
	default:
		if err := jpeg.Encode(&out, cropped, &jpeg.Options{Quality: 92}); err != nil {
			return imageBytes, contentType
		}
		slog.Info("kids_image_panel_crop_applied", "format", format, "width", rect.Dx(), "height", rect.Dy())
		return out.Bytes(), "image/jpeg"
	}
}

type axisSpan struct {
	start int
	end   int
}

func detectLargestPanelRect(img image.Image) (image.Rectangle, bool) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < 240 || height < 240 {
		return image.Rectangle{}, false
	}

	colGutters := detectColumnGutters(img, bounds)
	rowGutters := detectRowGutters(img, bounds)
	xSpans := nonGutterSpans(colGutters, max(60, width/8))
	ySpans := nonGutterSpans(rowGutters, max(60, height/8))
	if len(xSpans) < 2 || len(ySpans) < 2 {
		return image.Rectangle{}, false
	}

	imgCenterX := float64(bounds.Min.X + width/2)
	imgCenterY := float64(bounds.Min.Y + height/2)
	bestScore := -1
	bestDist := math.MaxFloat64
	bestRect := image.Rectangle{}

	for _, xs := range xSpans {
		for _, ys := range ySpans {
			rect := image.Rect(bounds.Min.X+xs.start, bounds.Min.Y+ys.start, bounds.Min.X+xs.end, bounds.Min.Y+ys.end)
			if rect.Dx() < max(60, width/8) || rect.Dy() < max(60, height/8) {
				continue
			}
			area := rect.Dx() * rect.Dy()
			centerX := float64(rect.Min.X+rect.Max.X) / 2
			centerY := float64(rect.Min.Y+rect.Max.Y) / 2
			dist := math.Pow(centerX-imgCenterX, 2) + math.Pow(centerY-imgCenterY, 2)
			if area > bestScore || (area == bestScore && dist < bestDist) {
				bestScore = area
				bestDist = dist
				bestRect = rect
			}
		}
	}

	if bestScore <= 0 || bestRect.Empty() {
		return image.Rectangle{}, false
	}
	if bestRect.Dx()*bestRect.Dy() >= (width*height*92)/100 {
		return image.Rectangle{}, false
	}
	return bestRect, true
}

func detectColumnGutters(img image.Image, bounds image.Rectangle) []bool {
	width := bounds.Dx()
	height := bounds.Dy()
	gutters := make([]bool, width)
	required := int(float64(height) * 0.94)
	for x := 0; x < width; x++ {
		whiteCount := 0
		for y := 0; y < height; y++ {
			if isNearWhite(img.At(bounds.Min.X+x, bounds.Min.Y+y)) {
				whiteCount++
			}
		}
		gutters[x] = whiteCount >= required
	}
	return gutters
}

func detectRowGutters(img image.Image, bounds image.Rectangle) []bool {
	width := bounds.Dx()
	height := bounds.Dy()
	gutters := make([]bool, height)
	required := int(float64(width) * 0.94)
	for y := 0; y < height; y++ {
		whiteCount := 0
		for x := 0; x < width; x++ {
			if isNearWhite(img.At(bounds.Min.X+x, bounds.Min.Y+y)) {
				whiteCount++
			}
		}
		gutters[y] = whiteCount >= required
	}
	return gutters
}

func nonGutterSpans(gutters []bool, minSpan int) []axisSpan {
	var spans []axisSpan
	start := -1
	for i, isGutter := range gutters {
		if !isGutter && start == -1 {
			start = i
		}
		if isGutter && start != -1 {
			if i-start >= minSpan {
				spans = append(spans, axisSpan{start: start, end: i})
			}
			start = -1
		}
	}
	if start != -1 && len(gutters)-start >= minSpan {
		spans = append(spans, axisSpan{start: start, end: len(gutters)})
	}
	return spans
}

func isNearWhite(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r>>8 >= 242 && g>>8 >= 242 && b>>8 >= 242
}

func cropImage(src image.Image, rect image.Rectangle) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(dst, dst.Bounds(), src, rect.Min, draw.Src)
	return dst
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func handleStart(engine *service.Engine, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.StartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		strictGenre := cfg.FeatureEnabled("strict_genre_enforcement")
		if err := validateStartRequest(&req, strictGenre); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		session, log, choices, err := engine.StartSession(r.Context(), &req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := map[string]interface{}{
			"session_id":     session.ID,
			"user_name":      session.UserName,
			"state":          session.State,
			"story":          log.Content,
			"syllable_split": log.ColorCodedContent,
			"choices":        choices,
			"status":         session.Status,
			"chapter_title":  log.ChapterTitle,
			"age":            session.Age,
			"age_group":      kidsAgeGroup(session.Genre, session.Age),
			"word_count":     service.CountWords(log.Content),
		}
		if domain.IsKidsGenre(session.Genre) {
			resp["page_number"] = log.TurnNumber
			resp["total_pages"] = service.KidsStoryPageCount
			resp["page_label"] = service.KidsPageIndicator(log.TurnNumber)
		}
		addKidsStorybookFields(resp, cfg, session, log)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func validateStartRequest(req *domain.StartRequest, strictGenreEnforcement bool) error {
	if req.Name == "" || req.Genre == "" {
		return errors.New("name and genre are required")
	}
	if !domain.IsValidGenre(req.Genre) {
		return errors.New("invalid genre selection")
	}
	// Archetype is optional for kids genres — defaults to "Hero" in the engine.
	if req.Archetype != "" {
		if !domain.IsValidArchetype(req.Archetype) {
			return errors.New("invalid archetype selection")
		}
		if strictGenreEnforcement {
			if !domain.IsValidArchetypeForGenre(req.Archetype, req.Genre) {
				return errors.New("archetype is not valid for the selected genre")
			}
		}
	}
	normalizeAge(req)
	if !domain.IsKidsGenre(req.Genre) {
		if req.Age == nil || *req.Age < 10 || *req.Age > 150 {
			return errors.New("age must be between 10 and 150")
		}
	}
	return nil
}

func normalizeAge(req *domain.StartRequest) {
	if domain.IsKidsGenre(req.Genre) {
		if req.Age == nil {
			age := domain.KidsDefaultAge
			req.Age = &age
			return
		}
		if *req.Age < domain.KidsMinAge || *req.Age > domain.KidsMaxAge {
			age := domain.KidsMinAge
			req.Age = &age
		}
		return
	}
	if req.Age == nil {
		age := 21
		req.Age = &age
	}
}

func kidsAgeGroup(genre string, age int) string {
	if !domain.IsKidsGenre(genre) {
		return ""
	}
	return domain.KidsAgeTier(age)
}

func addKidsStorybookFields(resp map[string]interface{}, cfg *config.Config, session *domain.Session, log *domain.StoryLog) {
	if session == nil || log == nil || !domain.IsKidsGenre(session.Genre) || !cfg.FeatureEnabled("kids_storybook_v2") {
		return
	}
	resp["image_prompt"] = service.BuildKidsImagePrompt(log.ImageScene, session.State.VisualSetting, session.State.Entities, session.Genre, session.Age)
	resp["image_url"] = service.KidsImageURL(log.ImageScene, session.State.VisualSetting, session.State.Entities, session.Genre, session.Age)
}

func handleTurn(engine *service.Engine, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string `json:"session_id"`
			ChoiceIdx int    `json:"choice_index"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		session, log, newChoices, err := engine.ProcessTurn(r.Context(), req.SessionID, req.ChoiceIdx)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidChoice) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := map[string]interface{}{
			"state":          session.State,
			"story":          log.Content,
			"syllable_split": log.ColorCodedContent,
			"choices":        newChoices,
			"status":         session.Status,
			"chapter_title":  log.ChapterTitle,
			"twists":         session.State.PlotTwists,
			"age":            session.Age,
			"age_group":      kidsAgeGroup(session.Genre, session.Age),
			"word_count":     service.CountWords(log.Content),
		}
		if domain.IsKidsGenre(session.Genre) {
			resp["page_number"] = log.TurnNumber
			resp["total_pages"] = service.KidsStoryPageCount
			resp["page_label"] = service.KidsPageIndicator(log.TurnNumber)
		}
		addKidsStorybookFields(resp, cfg, session, log)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func handleDownload(engine *service.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "sessionID")

		text, err := engine.GenerateStoryText(r.Context(), sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Disposition", "attachment; filename=story.txt")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(text))
	}
}

func handleGetSession(engine *service.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "sessionID")
		session, err := engine.GetSession(r.Context(), sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if session == nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session_id": session.ID,
			"user_name":  session.UserName,
			"state":      session.State,
			"status":     session.Status,
			"choices":    session.CurrentChoices,
			"age":        session.Age,
			"age_group":  kidsAgeGroup(session.Genre, session.Age),
		})
	}
}
