package transport

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
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
		if cfg.OpenAIKey == "" {
			http.Error(w, "image generation is not configured", http.StatusServiceUnavailable)
			return
		}
		model := strings.TrimSpace(cfg.OpenAIImageModel)
		if model == "" {
			model = "gpt-image-1"
		}

		payload := map[string]interface{}{
			"model":         model,
			"prompt":        prompt,
			"n":             1,
			"size":          "auto",
			"quality":       "auto",
			"background":    "auto",
			"image_detail":  "high",
			"output_format": "png",
		}
		body, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, "failed to build image request", http.StatusInternalServerError)
			return
		}

		base := strings.TrimRight(cfg.OpenAIBase, "/")
		if base == "" {
			base = "https://llm.code-x.my/v1"
		}
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, base+"/images/generations?response_format=binary", bytes.NewReader(body))
		if err != nil {
			http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)
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

		imageBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "failed to read image", http.StatusBadGateway)
			return
		}
		pngBytes := ensurePNG(imageBytes)
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "private, max-age=86400")
		_, _ = w.Write(pngBytes)
	}
}

func ensurePNG(imageBytes []byte) []byte {
	if len(imageBytes) >= 8 && bytes.Equal(imageBytes[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return imageBytes
	}
	if len(imageBytes) >= 2 && imageBytes[0] == 0xff && imageBytes[1] == 0xd8 {
		img, _, err := image.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return imageBytes
		}
		var out bytes.Buffer
		if err := png.Encode(&out, img); err != nil {
			return imageBytes
		}
		return out.Bytes()
	}
	return imageBytes
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
	resp["character_profiles"] = session.State.Entities
	resp["image_prompt"] = service.BuildKidsImagePrompt(log.ImageScene, session.State.Entities, session.Genre, session.Age)
	resp["image_url"] = service.KidsImageURL(log.ImageScene, session.State.Entities, session.Genre, session.Age)
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
			"state":      session.State,
			"status":     session.Status,
			"choices":    session.CurrentChoices,
			"age":        session.Age,
			"age_group":  kidsAgeGroup(session.Genre, session.Age),
		})
	}
}
