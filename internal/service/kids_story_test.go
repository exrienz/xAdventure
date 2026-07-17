package service

import (
	"strings"
	"testing"

	"github.com/muz/xadventure/internal/domain"
)

func TestKidsStyleRulesIncludesStrictBahasaMalaysiaAndAgeBounds(t *testing.T) {
	prompt := KidsStyleRules("Pengembaraan", 4, true)

	for _, want := range []string{
		"standard Bahasa Malaysia only",
		"Indonesian",
		"20 words",
		"age 4-5",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestCountWordsStripsHTML(t *testing.T) {
	got := CountWords(`<span style="color:black;">Bu</span><span style="color:red;">ku</span> ini merah.`)
	if got != 3 {
		t.Fatalf("CountWords = %d; want 3", got)
	}
}

func TestIndonesianMarkers(t *testing.T) {
	markers := IndonesianMarkers("Gue mau beli sepeda dan bilang gak mau pergi.")
	if len(markers) == 0 {
		t.Fatal("expected Indonesian markers")
	}
	for _, want := range []string{"gue", "mau", "sepeda", "gak/nggak"} {
		if !contains(markers, want) {
			t.Fatalf("markers %v missing %q", markers, want)
		}
	}
}

func TestIndonesianMarkersAllowsBahasaMalaysia(t *testing.T) {
	markers := IndonesianMarkers("Saya mahu membaca buku di sekolah dengan kawan-kawan.")
	if len(markers) != 0 {
		t.Fatalf("unexpected Indonesian markers: %v", markers)
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func TestKidsWordCapsMatchRequirement(t *testing.T) {
	tests := map[int]int{4: 20, 5: 20, 6: 40, 7: 40, 8: 80}
	for age, want := range tests {
		if got := KidsMaxWords(age); got != want {
			t.Fatalf("KidsMaxWords(%d) = %d; want %d", age, got, want)
		}
	}
}

func TestKidsStyleRulesIncludesAgeVocabularyGuidance(t *testing.T) {
	tests := []struct {
		age  int
		want string
	}{
		{4, "20 words"},
		{5, "very simple vocabulary"},
		{6, "40 words"},
		{7, "simple sentences"},
		{8, "80 words"},
	}
	for _, tt := range tests {
		prompt := KidsStyleRules("Pengembaraan", tt.age, true)
		if !strings.Contains(prompt, tt.want) {
			t.Fatalf("KidsStyleRules(%d) missing %q:\n%s", tt.age, tt.want, prompt)
		}
	}
}

func TestEnforceKidsWordLimitTruncates(t *testing.T) {
	words := make([]string, 90)
	for i := range words {
		words[i] = "kata"
	}
	got := EnforceKidsWordLimit(strings.Join(words, " "), 6)
	if CountWords(got) != 40 {
		t.Fatalf("truncated word count = %d; want 40", CountWords(got))
	}
}

func TestKidsImageURLUsesPollinationsAndSanitizesPrompt(t *testing.T) {
	url := KidsImageURL(`<script>alert(1)</script> Ali jumpa kucing!`, nil, "Fantasi", 6)
	_ = url // still test sanitization
	if !strings.HasPrefix(url, "/api/kids/image?prompt=") {
		t.Fatalf("unexpected image url: %s", url)
	}
	if strings.Contains(url, "<script>") || strings.Contains(url, "alert") {
		t.Fatalf("unsafe prompt leaked into url: %s", url)
	}
}

func TestKidsImagePromptUsesCharacterAppearanceNotFullStoryDump(t *testing.T) {
	imageScene := "A young girl with long dark hair, pink shirt, and blue skirt kneels beside a small orange kitten under a big tree. The kitten looks scared. Green grass, butterflies, cool breeze."
	entities := map[string]domain.Entity{
		"naila": {
			Name:       "Naila",
			Role:       "protagonist",
			Appearance: "young Malaysian girl with long dark hair, pink shirt, blue skirt, white shoes, yellow butterfly hair clip",
			Traits:     []string{"kind", "curious"},
		},
		"kucing": {
			Name:       "Mimi",
			Role:       "side character kitten",
			Appearance: "small orange kitten with white paws and a tiny blue ribbon",
			Traits:     []string{"timid"},
		},
	}

	prompt := BuildKidsImagePrompt(imageScene, entities, "Fantasi", 4)
	for _, want := range []string{"long dark hair", "pink shirt", "blue skirt", "yellow butterfly hair clip", "small orange kitten", "Scene:"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing visual detail %q:\n%s", want, prompt)
		}
	}
	// Must NOT contain Malay story text
	for _, bad := range []string{"bermain", "mengejar", "rama-rama", "ketakutan"} {
		if strings.Contains(prompt, bad) {
			t.Fatalf("prompt contains Malay text %q:\n%s", bad, prompt)
		}
	}
}
