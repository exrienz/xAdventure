package transport

import (
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
	req := &domain.StartRequest{Genre: "Fabel", Age: &age}
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
		Name:      "Ali",
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
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("PNGDATA"))
	}))
	defer upstream.Close()

	cfg := &config.Config{
		OpenAIBase:       upstream.URL + "/v1",
		OpenAIKey:        "test-key",
		OpenAIImageModel: "gpt-image-test",
		LLMTimeoutSec:    5,
		FeatureFlags:     map[string]bool{"kids_mode": true},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/kids/image?prompt="+strings.ReplaceAll("girl and cat", " ", "+"), nil)
	rec := httptest.NewRecorder()

	handleKidsImage(cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if gotPath != "/v1/images/generations?response_format=binary" {
		t.Fatalf("upstream path = %q", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if gotPayload["model"] != "gpt-image-test" || gotPayload["prompt"] != "girl and cat" || gotPayload["output_format"] != "png" {
		t.Fatalf("unexpected payload: %#v", gotPayload)
	}
	if rec.Header().Get("Content-Type") != "image/png" || rec.Body.String() != "PNGDATA" {
		t.Fatalf("unexpected response: type=%q body=%q", rec.Header().Get("Content-Type"), rec.Body.String())
	}
}
