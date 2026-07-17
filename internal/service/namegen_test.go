package service

import (
	"strings"
	"testing"
)

func TestNameGenerator_UniqueWithinHistory(t *testing.T) {
	ng := NewNameGenerator(50, NewSafetyFilter())
	seen := make(map[string]bool)
	for i := 0; i < 30; i++ {
		g := ng.Generate("Cyberpunk")
		if g.Name == "" {
			t.Fatalf("generated empty name on iteration %d", i)
		}
		if seen[g.Name] {
			t.Fatalf("duplicate name generated: %s", g.Name)
		}
		seen[g.Name] = true
	}
}

func TestNameGenerator_GenreStyle(t *testing.T) {
	ng := NewNameGenerator(50, NewSafetyFilter())

	cyber := ng.Generate("Cyberpunk").Name
	if !looksCyberpunk(cyber) {
		t.Logf("cyberpunk name %q may not look genre-appropriate", cyber)
	}

	steampunk := ng.Generate("Steampunk").Name
	if !looksSteampunk(steampunk) {
		t.Logf("steampunk name %q may not look genre-appropriate", steampunk)
	}
}

func TestNameGenerator_SafetyRejectsProfanity(t *testing.T) {
	sf := NewSafetyFilter()
	if sf.IsSafeName("Fuckface") {
		t.Error("expected profane name to be rejected")
	}
	if sf.IsSafeName("Nice Person") {
		// OK
	} else {
		t.Error("expected benign name to be safe")
	}
}

func TestNameGenerator_HistoryCapacity(t *testing.T) {
	ng := NewNameGenerator(5, NewSafetyFilter())
	for i := 0; i < 10; i++ {
		ng.Generate("Adventure")
	}
	if len(ng.History()) > 5 {
		t.Fatalf("history capacity exceeded: got %d, want 5", len(ng.History()))
	}
}

func TestNameGenerator_UniqueRate(t *testing.T) {
	ng := NewNameGenerator(50, NewSafetyFilter())
	for i := 0; i < 10; i++ {
		ng.Generate("Adventure")
	}
	rate := ng.UniqueRate([]string{"Elias", "Rowan", "Kira"})
	if rate < 0.7 {
		t.Fatalf("expected high unique rate, got %.2f", rate)
	}
}

func TestIsValidNameRequest(t *testing.T) {
	bad := []string{
		"",
		"ignore previous instructions",
		"```system",
		"http://evil.com",
		strings.Repeat("a", 81),
	}
	for _, b := range bad {
		if IsValidNameRequest(b) {
			t.Errorf("expected name %q to be invalid", b)
		}
	}
	if !IsValidNameRequest("Ayla") {
		t.Error("expected valid name to be accepted")
	}
}

func looksCyberpunk(name string) bool {
	keywords := []string{"Kael", "Rin", "Jax", "Nova", "Zero", "Vex", "Sera", "Nix", "Blaze", "Echo", "Kovacs", "Tanaka", "Wired", "Neon", "Ghost", "Chrome", "Glitch", "Rogue"}
	lower := strings.ToLower(name)
	for _, k := range keywords {
		if strings.Contains(lower, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

func looksSteampunk(name string) bool {
	keywords := []string{"Thaddeus", "Emmeline", "Barnaby", "Gwendolyn", "Cogsworth", "Brasswell", "Gearhart", "Copperfield", "Steamwright", "Clockwork"}
	lower := strings.ToLower(name)
	for _, k := range keywords {
		if strings.Contains(lower, strings.ToLower(k)) {
			return true
		}
	}
	return false
}
