package service

import (
	"testing"

	"github.com/muz/xadventure/internal/config"
	"github.com/muz/xadventure/internal/llm"
)

func TestEngineTextClientForGenreUsesKidsProviderWhenPresent(t *testing.T) {
	defaultClient := llm.NewClient("https://default.example/v1", "default-key", "default-model", 10, 0, 0.8, 0.9)
	kidsClient := llm.NewClient("https://kids.example/v1", "kids-key", "kids-model", 10, 0, 0.8, 0.9)

	engine := NewEngine(nil, defaultClient, kidsClient, &config.Config{})

	if got := engine.textClientForGenre("Pengembaraan"); got != kidsClient {
		t.Fatal("expected kids genre to use kids client when configured")
	}
	if got := engine.textClientForGenre("Adventure"); got != defaultClient {
		t.Fatal("expected non-kids genre to use default client")
	}
}

func TestEngineTextClientForGenreFallsBackWithoutKidsProvider(t *testing.T) {
	defaultClient := llm.NewClient("https://default.example/v1", "default-key", "default-model", 10, 0, 0.8, 0.9)

	engine := NewEngine(nil, defaultClient, nil, &config.Config{})

	if got := engine.textClientForGenre("Fantasi"); got != defaultClient {
		t.Fatal("expected kids genre to fall back to default client when kids provider is absent")
	}
}
