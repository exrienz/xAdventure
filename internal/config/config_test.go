package config

import (
	"os"
	"testing"
)

func TestIsValidModelNameAllowsCommonProviderFormats(t *testing.T) {
	valid := []string{
		"Malay",
		"gpt-4.1-mini",
		"openai/gpt-4o-mini",
		"llama3.1:8b",
		"provider/model:tag-1.2",
	}
	for _, name := range valid {
		if !isValidModelName(name) {
			t.Fatalf("expected %q to be valid", name)
		}
	}
}

func TestIsValidModelNameRejectsUnsafeValues(t *testing.T) {
	invalid := []string{"", "bad model", "../secret", "model$", "model?x=1"}
	for _, name := range invalid {
		if isValidModelName(name) {
			t.Fatalf("expected %q to be invalid", name)
		}
	}
}

func TestLoadProviderDefaultsAndOverrides(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("OPENAI_API_BASE", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("KIDS_LLM_API_BASE", "https://kids.example/v1")
	t.Setenv("KIDS_LLM_API_KEY", "kids-key")
	t.Setenv("KIDS_LLM_MODEL", "kids-model")
	t.Setenv("IMAGEROUTER_API_BASE", "")
	t.Setenv("IMAGEROUTER_API_KEY", "img-key")
	t.Setenv("IMAGEROUTER_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.OpenAIBase != "https://api.openai.com/v1" || cfg.OpenAIModel != "gpt-4o" {
		t.Fatalf("unexpected default openai config: %#v", cfg)
	}
	if cfg.KidsLLMBase != "https://kids.example/v1" || cfg.KidsLLMKey != "kids-key" || cfg.KidsLLMModel != "kids-model" {
		t.Fatalf("unexpected kids llm config: %#v", cfg)
	}
	if cfg.ImageRouterBase != "https://api.imagerouter.io/v1/openai" || cfg.ImageRouterKey != "img-key" || cfg.ImageRouterModel != "imagerouter/auto-image" {
		t.Fatalf("unexpected imagerouter config: %#v", cfg)
	}
}

func TestProviderPresenceHelpers(t *testing.T) {
	cfg := &Config{}
	if cfg.HasKidsLLMProvider() || cfg.HasImageRouterProvider() {
		t.Fatal("empty config should not report providers configured")
	}

	cfg.KidsLLMBase = "https://kids.example/v1"
	cfg.KidsLLMKey = "kids-key"
	cfg.KidsLLMModel = "kids-model"
	cfg.ImageRouterBase = "https://api.imagerouter.io/v1/openai"
	cfg.ImageRouterKey = "img-key"
	cfg.ImageRouterModel = "imagerouter/auto-image"

	if !cfg.HasKidsLLMProvider() {
		t.Fatal("expected kids llm provider to be configured")
	}
	if !cfg.HasImageRouterProvider() {
		t.Fatal("expected image router provider to be configured")
	}
}

func TestLoadIgnoresInvalidKidsModelName(t *testing.T) {
	orig := os.Getenv("KIDS_LLM_MODEL")
	t.Cleanup(func() {
		_ = os.Setenv("KIDS_LLM_MODEL", orig)
	})
	t.Setenv("KIDS_LLM_MODEL", "bad model")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.KidsLLMModel != "" {
		t.Fatalf("expected invalid kids model name to be cleared, got %q", cfg.KidsLLMModel)
	}
}
