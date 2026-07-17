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
		"at least 6 pages and at most 10 pages",
		"15-30 words",
		"Age 4",
		"Aim for about",
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
	tests := map[int]int{4: 30, 5: 50, 6: 80, 7: 120, 8: 180}
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
		{4, "15-30 words"},
		{5, "simple familiar vocabulary"},
		{6, "40-80 words"},
		{7, "70-120 words"},
		{8, "100-180 words"},
	}
	for _, tt := range tests {
		prompt := KidsStyleRules("Pengembaraan", tt.age, true)
		if !strings.Contains(prompt, tt.want) {
			t.Fatalf("KidsStyleRules(%d) missing %q:\n%s", tt.age, tt.want, prompt)
		}
	}
}

func TestEnforceKidsWordLimitTruncates(t *testing.T) {
	words := make([]string, 190)
	for i := range words {
		words[i] = "kata"
	}
	got := EnforceKidsWordLimit(strings.Join(words, " "), 6)
	if CountWords(got) > 80 {
		t.Fatalf("truncated word count = %d; want <= 80", CountWords(got))
	}
	if strings.HasSuffix(strings.TrimSpace(got), "kata") && !strings.HasSuffix(strings.TrimSpace(got), ".") {
		t.Fatalf("expected sentence-safe ending, got %q", got)
	}
}

func TestEnforceKidsWordLimitRemovesIncompleteTrailingFragment(t *testing.T) {
	text := "Nuha duduk di atas rumput yang lembut. Dia lihat awan putih di langit biru. Tiba-tiba, Nuha nampak kilauan berwarna pelangi di dalam hutan kecil. Nuha rasa ingin tahu sangat. Dia"
	got := EnforceKidsWordLimit(text, 5)
	if strings.HasSuffix(strings.TrimSpace(got), "Dia") {
		t.Fatalf("expected incomplete fragment to be removed, got %q", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), ".") {
		t.Fatalf("expected complete sentence ending, got %q", got)
	}
}

func TestKidsPageInstructionBuildsNaturalOpeningAndEnding(t *testing.T) {
	opening := KidsPageInstruction(1, 5)
	for _, want := range []string{"PAGE 1 OF 10", "Start at the true beginning", "25-50 words"} {
		if !strings.Contains(opening, want) {
			t.Fatalf("opening instruction missing %q:\n%s", want, opening)
		}
	}

	ending := KidsPageInstruction(10, 5)
	for _, want := range []string{"PAGE 10 OF 10", "final page", "happy and satisfying ending", "Do NOT end with only a realization"} {
		if !strings.Contains(ending, want) {
			t.Fatalf("ending instruction missing %q:\n%s", want, ending)
		}
	}

	earliestEnding := KidsPageInstruction(6, 5)
	for _, want := range []string{"PAGE 6 OF 10", "may end naturally", "story_complete", "Do NOT save the ending for page 10"} {
		if !strings.Contains(earliestEnding, want) {
			t.Fatalf("page 6 instruction missing %q:\n%s", want, earliestEnding)
		}
	}
}

func TestKidsPageIndicator(t *testing.T) {
	if got := KidsPageIndicator(1); got != "Halaman 1 / maks 10" {
		t.Fatalf("KidsPageIndicator(1) = %q", got)
	}
	if got := KidsPageIndicator(99); got != "Halaman 10 / maks 10" {
		t.Fatalf("KidsPageIndicator(99) = %q", got)
	}
}

func TestKidsImageURLUsesLocalProxyAndSanitizesPrompt(t *testing.T) {
	url := KidsImageURL(`<script>alert(1)</script> Watak jumpa kucing!`, "sunny village field with a mango tree", nil, "Fantasi", 6)
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
	visualSetting := "sunny village field with a mango tree, wooden fence, soft morning light, and warm watercolor atmosphere"
	entities := map[string]domain.Entity{
		"watak-utama": {
			Name:       "WatakUtama",
			Role:       "protagonist",
			Appearance: "young Malaysian girl with long dark hair, pink shirt, blue skirt, white shoes, yellow butterfly hair clip",
			Traits:     []string{"kind", "curious"},
		},
		"watak-sampingan": {
			Name:       "KawanKecil",
			Role:       "side character kitten",
			Appearance: "small orange kitten with white paws and a tiny blue ribbon",
			Traits:     []string{"timid"},
		},
	}

	prompt := BuildKidsImagePrompt(imageScene, visualSetting, entities, "Fantasi", 4)
	for _, want := range []string{"long dark hair", "pink shirt", "blue skirt", "yellow butterfly hair clip", "small orange kitten", "Story setting:", "mango tree", "Current page scene:", "single full-page illustration", "No split panels"} {
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
