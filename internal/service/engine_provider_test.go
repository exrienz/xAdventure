package service

import (
	"strings"
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

func TestDefaultKidsAppearanceUsesKnownCharacterProfileInWatakMode(t *testing.T) {
	got := defaultKidsAppearance("SpongeBob", "Watak")
	for _, want := range []string{"yellow sea sponge", "square body", "red tie", "brown square shorts"} {
		if !strings.Contains(got, want) {
			t.Fatalf("appearance missing %q: %s", want, got)
		}
	}
}

func TestDefaultKidsAppearanceFallsBackToRecognizableCharacterHint(t *testing.T) {
	got := defaultKidsAppearance("TotoroX", "Watak")
	if !strings.Contains(got, "recognizable fictional character") {
		t.Fatalf("expected recognizable character fallback, got %s", got)
	}
	if !strings.Contains(got, "TotoroX") {
		t.Fatalf("expected name to appear in fallback, got %s", got)
	}
}

func TestKidsSettingSparkIsDeterministicForSameSeed(t *testing.T) {
	a := KidsSettingSpark("Fantasi", "abc123")
	b := KidsSettingSpark("Fantasi", "abc123")
	if a != b {
		t.Fatalf("expected deterministic setting spark, got %q vs %q", a, b)
	}
}

func TestKidsSettingSparkVariesByGenre(t *testing.T) {
	a := KidsSettingSpark("Fantasi", "same-seed")
	b := KidsSettingSpark("Sains & Teknologi", "same-seed")
	if a == b {
		t.Fatalf("expected genre-specific variation, both were %q", a)
	}
}
