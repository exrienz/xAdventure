package transport

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/muz/xadventure/internal/config"
	"github.com/muz/xadventure/internal/domain"
)

func TestNormalizeAgeKidsMissingDefaultsToStandardLengthTier(t *testing.T) {
	req := &domain.StartRequest{Genre: "Pengembaraan"}
	normalizeAge(req)

	if req.Age == nil || *req.Age != domain.KidsDefaultAge {
		t.Fatalf("missing kids age = %v; want %d", req.Age, domain.KidsDefaultAge)
	}
}

func TestNormalizeAgeKidsInvalidDefaultsToLowestTier(t *testing.T) {
	age := 13
	req := &domain.StartRequest{Genre: "Kisah Dongeng", Age: &age}
	normalizeAge(req)

	if req.Age == nil || *req.Age != domain.KidsMinAge {
		t.Fatalf("invalid kids age = %v; want %d", req.Age, domain.KidsMinAge)
	}
}

func TestNormalizeAgeStandardMissingDefaultsToAdultAge(t *testing.T) {
	req := &domain.StartRequest{Genre: "Adventure"}
	normalizeAge(req)

	if req.Age == nil || *req.Age != 21 {
		t.Fatalf("missing standard age = %v; want 21", req.Age)
	}
}

func TestValidateStartRequestKidsInvalidAgeIsClamped(t *testing.T) {
	age := 2
	req := &domain.StartRequest{
		Name:      "WatakUjian",
		Age:       &age,
		Gender:    "Lelaki",
		Genre:     "Cerita Haiwan",
		Archetype: "Hero",
	}

	if err := validateStartRequest(req, true); err != nil {
		t.Fatalf("expected invalid kids age to clamp, got %v", err)
	}
	if req.Age == nil || *req.Age != domain.KidsMinAge {
		t.Fatalf("age was not clamped to lowest tier: %v", req.Age)
	}
}

func TestHandleKidsImagePostsToConfiguredImageEndpoint(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotPayload map[string]interface{}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.RequestURI()
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotPayload); err != nil {
			t.Fatalf("invalid request json: %v", err)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("JPEGDATA"))
	}))
	defer upstream.Close()

	cfg := &config.Config{
		ImageRouterBase:  upstream.URL + "/v1/openai",
		ImageRouterKey:   "test-key",
		ImageRouterModel: "imagerouter/auto-image",
		LLMTimeoutSec:    5,
		FeatureFlags:     map[string]bool{"kids_mode": true},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/kids/image?prompt="+strings.ReplaceAll("girl and cat", " ", "+"), nil)
	rec := httptest.NewRecorder()

	handleKidsImage(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if gotPath != "/v1/openai/images/generations" {
		t.Fatalf("upstream path = %q", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if gotPayload["model"] != "imagerouter/auto-image" ||
		gotPayload["prompt"] != "girl and cat" ||
		gotPayload["quality"] != "low" ||
		gotPayload["size"] != "1024x1024" ||
		gotPayload["output_format"] != "jpeg" {
		t.Fatalf("unexpected payload: %#v", gotPayload)
	}
	if rec.Header().Get("Content-Type") != "image/jpeg" || rec.Body.String() != "JPEGDATA" {
		t.Fatalf("unexpected response: type=%q body=%q", rec.Header().Get("Content-Type"), rec.Body.String())
	}
}

func TestHandleKidsImageDecodesOpenAIStyleJSONImagePayload(t *testing.T) {
	var gotPath string
	var gotPayload map[string]interface{}
	imageBytes := []byte("JPEGDATA")
	imageB64 := base64.StdEncoding.EncodeToString(imageBytes)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotPayload); err != nil {
			t.Fatalf("invalid request json: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"` + imageB64 + `"}]}`))
	}))
	defer upstream.Close()

	cfg := &config.Config{
		ImageRouterBase:  upstream.URL + "/v1/openai",
		ImageRouterKey:   "test-key",
		ImageRouterModel: "imagerouter/auto-image",
		LLMTimeoutSec:    5,
		FeatureFlags:     map[string]bool{"kids_mode": true},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/kids/image?prompt="+strings.ReplaceAll("girl and cat", " ", "+"), nil)
	rec := httptest.NewRecorder()

	handleKidsImage(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if gotPath != "/v1/openai/images/generations" {
		t.Fatalf("upstream path = %q", gotPath)
	}
	if gotPayload["model"] != "imagerouter/auto-image" {
		t.Fatalf("unexpected payload: %#v", gotPayload)
	}
	if rec.Header().Get("Content-Type") != "image/jpeg" || rec.Body.String() != string(imageBytes) {
		t.Fatalf("unexpected response: type=%q body=%q", rec.Header().Get("Content-Type"), rec.Body.String())
	}
}

func TestHandleKidsImageReturns503WhenImageRouterMissing(t *testing.T) {
	cfg := &config.Config{
		FeatureFlags: map[string]bool{"kids_mode": true},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/kids/image?prompt=test", nil)
	rec := httptest.NewRecorder()

	handleKidsImage(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestNormalizeKidsStoryImageCropsComicGridToLargestPanel(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 600, 600))
	white := color.RGBA{255, 255, 255, 255}
	red := color.RGBA{220, 40, 40, 255}
	blue := color.RGBA{40, 120, 220, 255}
	green := color.RGBA{40, 180, 90, 255}

	fillRect(img, img.Bounds(), white)
	fillRect(img, image.Rect(20, 20, 180, 180), red)
	fillRect(img, image.Rect(220, 20, 380, 180), red)
	fillRect(img, image.Rect(420, 20, 580, 180), red)
	fillRect(img, image.Rect(20, 220, 180, 380), red)
	fillRect(img, image.Rect(220, 220, 520, 520), blue)
	fillRect(img, image.Rect(20, 420, 180, 580), red)
	fillRect(img, image.Rect(420, 220, 580, 380), green)
	fillRect(img, image.Rect(220, 540, 380, 580), green)
	fillRect(img, image.Rect(420, 420, 580, 580), green)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
		t.Fatalf("encode test image: %v", err)
	}

	out, contentType := normalizeKidsStoryImage(buf.Bytes(), "image/jpeg")
	if contentType != "image/jpeg" {
		t.Fatalf("content type = %q; want image/jpeg", contentType)
	}

	cropped, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode cropped image: %v", err)
	}
	if cropped.Bounds().Dx() >= 600 || cropped.Bounds().Dy() >= 600 {
		t.Fatalf("expected cropped image smaller than original, got %dx%d", cropped.Bounds().Dx(), cropped.Bounds().Dy())
	}
	center := color.RGBAModel.Convert(cropped.At(cropped.Bounds().Dx()/2, cropped.Bounds().Dy()/2)).(color.RGBA)
	if center.B <= center.R {
		t.Fatalf("expected crop to keep largest blue panel, got center color %#v", center)
	}
}

func fillRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}
