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
	got := defaultKidsProfile("SpongeBob", "Watak", "Fantasi", "seed-1").Appearance
	for _, want := range []string{"yellow sea sponge", "square body", "red tie", "brown square shorts"} {
		if !strings.Contains(got, want) {
			t.Fatalf("appearance missing %q: %s", want, got)
		}
	}
}

func TestDefaultKidsAppearanceFallsBackToRecognizableCharacterHint(t *testing.T) {
	got := defaultKidsProfile("TotoroX", "Watak", "Fantasi", "seed-1").Appearance
	if !strings.Contains(got, "recognizable fictional character") {
		t.Fatalf("expected recognizable character fallback, got %s", got)
	}
	if !strings.Contains(got, "TotoroX") {
		t.Fatalf("expected name to appear in fallback, got %s", got)
	}
}

func TestDefaultKidsProfileVariesByStorySeed(t *testing.T) {
	a := defaultKidsProfile("Alya", "Perempuan", "Fantasi", "seed-a")
	b := defaultKidsProfile("Alya", "Perempuan", "Fantasi", "seed-b")
	if a.Appearance == b.Appearance {
		t.Fatalf("expected different appearance across stories, got %q", a.Appearance)
	}
	if strings.Join(a.Traits, ",") == strings.Join(b.Traits, ",") {
		t.Fatalf("expected different traits across stories, got %v", a.Traits)
	}
}

func TestDefaultKidsProfileIsStableForSameStorySeed(t *testing.T) {
	a := defaultKidsProfile("Amir", "Lelaki", "Pengembaraan", "same-seed")
	b := defaultKidsProfile("Amir", "Lelaki", "Pengembaraan", "same-seed")
	if a.Appearance != b.Appearance {
		t.Fatalf("expected stable appearance, got %q vs %q", a.Appearance, b.Appearance)
	}
	if strings.Join(a.Traits, ",") != strings.Join(b.Traits, ",") {
		t.Fatalf("expected stable traits, got %v vs %v", a.Traits, b.Traits)
	}
}

func TestDefaultKidsProfileDoesNotUseLegacyFixedAppearance(t *testing.T) {
	got := defaultKidsProfile("Nuha", "Perempuan", "Fantasi", "seed-z").Appearance
	legacy := "young Malaysian girl with shoulder-length dark hair, yellow short-sleeve shirt, blue skirt, white socks, red shoes"
	if got == legacy {
		t.Fatalf("expected a generated profile instead of legacy fixed appearance")
	}
}

func TestMainGoalSummaryReadsGoalFlag(t *testing.T) {
	got := mainGoalSummary([]string{"visited_garden", "main_goal:find the missing moon seed"})
	if got != "find the missing moon seed" {
		t.Fatalf("mainGoalSummary = %q", got)
	}
}

func TestKidsStoryFlowInstructionCarriesCurrentGoal(t *testing.T) {
	got := kidsStoryFlowInstruction(4, []string{"main_goal:return the lantern to nenek"})
	for _, want := range []string{"Current main goal: return the lantern to nenek", "Do NOT invent a second main problem", "Do NOT wander into unrelated subplots"} {
		if !strings.Contains(got, want) {
			t.Fatalf("kidsStoryFlowInstruction missing %q: %s", want, got)
		}
	}
}

func TestKidsStoryFlowInstructionEstablishesGoalWhenMissing(t *testing.T) {
	got := kidsStoryFlowInstruction(1, nil)
	for _, want := range []string{"state_update.add_flags", "main_goal:<short ENGLISH phrase>", "Do NOT replace it with a new unrelated mission"} {
		if !strings.Contains(got, want) {
			t.Fatalf("kidsStoryFlowInstruction missing %q: %s", want, got)
		}
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
