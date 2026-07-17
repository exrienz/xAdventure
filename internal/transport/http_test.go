package transport

import (
	"encoding/base64"
	"encoding/json"
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
