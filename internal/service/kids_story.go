package service

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/muz/xadventure/internal/domain"
)

func KidsStyleRules(genre string, age int, dynamicKidsEnabled bool) string {
	tier := domainKidsAgeTier(age)
	prompt := `Style Rules (Strict Bahasa Malaysia for Kids):
- Write the ENTIRE story_text and choices in standard Bahasa Malaysia only.
- Use standard Malaysian Malay vocabulary and spelling. Do NOT use Indonesian dialect, slang, or syntax.
- STRICTLY FORBIDDEN Indonesian words (use the Malaysian equivalent instead):
  gak/nggak → tidak    | banget → sangat        | gue/lu → saya/kamu   | ngapain → buat apa
  mau → mahu           | uang → wang            | sepeda → basikal     | apa kabar → apa khabar
  rumah sakit → hospital | bego → bodoh         | jelek → buruk        | dong/sih → (omit)
  ngerti → faham       | bikin → buat           | ngerasa → rasa       | nulis → tulis
  kasih → beri         | bilang → kata          | aja → saja           | udah → sudah
  belom → belum        | kalo → kalau           | emang → memang       | kayak → macam
  gini → begini        | gitu → begitu          | trus → terus         | dimana → di mana
  gimana → bagaimana   | pake → pakai           | gede → besar         | matiin → matikan
  buka → buka          | dapet → dapat          | nyari → cari         | liat → lihat
  denger → dengar      | bener → betul          | rusak → rosak        | bagusin → baikkan
  tolongin → tolong    | bantuin → bantu        | ajarin → ajar        | nanya → tanya
  ceritain → cerita    | taruh → letak          | masukin → masukkan   | keluarin → keluarkan
  naikin → naikkan     | turunin → turunkan     | lewatin → lalui     | sampe → sampai
  deket → dekat        | jauhin → jauhkan       | pindahin → pindahkan | pulangin → pulangkan
  datangin → datangkan | pergiin → pergi        | balikin → kembalikan  | tungguin → tunggu
  mulaiin → mulaikan   | selesain → selesaikan  | habisin → habiskan   | lanjutin → sambung
  berhentiin → hentikan | ulangin → ulang       | cobain → cuba        | ngasih → beri
  makasih → terima kasih | karna → kerana       | biarin → biar        | nyampe → sampai
  nyobain → cuba       | nyokap → mak           | bokap → ayah         | halu → khayal
  lebay → berlebihan   | cupu → ketinggalan     | norak → kampungan    | alay → norak
- Use Malaysian vocabulary: cikgu, murid, tandas, kereta api, basikal, kemeja, setem, tiket,
  kerusi, meja, almari, katil, bantal, selimut, tingkap, peti sejuk, mesin basuh, lampu, suis,
  motosikal, telefon, televisyen, wayar, plag, mentol, kasut, stoking, seluar, songkok, tudung.
- The output MUST be 100% Bahasa Malaysia. Mixing with Bahasa Indonesia is NEVER acceptable.
- If you catch yourself using an Indonesian word, replace it with the Malaysian equivalent immediately.
- Keep the story child-safe, warm, playful, and educational.
- Do NOT include horror, explicit violence, romance, adult themes, or scary imagery.
- Keep sentences EXTREMELY short and simple. One clear idea per sentence.
- NO large walls of text or complex paragraphs.
- End every turn with a simple, age-appropriate hook or question.
- STORY ARC: The story runs for exactly 5 turns. Turn 1 = Pengenalan (Introduction), Turn 2-3 = Perkembangan (Development), Turn 4 = Klimaks (Climax), Turn 5 = Penyelesaian (Resolution, story ends).
- Each turn MUST advance the story meaningfully toward the natural conclusion. Do NOT stall or repeat.
- On Turn 5, the story MUST end completely with a happy, satisfying conclusion and a clear moral lesson. Never leave the story unfinished.`

	if !dynamicKidsEnabled {
		return prompt
	}

	minWords, maxWords := kidsWordBounds(age)
	return fmt.Sprintf(`%s

Dynamic Age-Based Length (%s):
- HARD LIMIT: story_text MUST be %d-%d words. DO NOT EXCEED %d words.
- For age 4-5: maximum 20 words, very simple vocabulary, tiny sentences, one clear action.
- For age 6-7: maximum 40 words, simple sentences, familiar vocabulary, light cause-and-effect.
- For age 8 or above: maximum 80 words, slightly richer vocabulary, still clear and child-safe.
- Choices must also follow the same language, tone, and simplicity rules.`, prompt, tier, minWords, maxWords, maxWords)
}

func KidsDynamicAgeInstruction(age int) string {
	tier := domainKidsAgeTier(age)
	minWords, maxWords := kidsWordBounds(age)
	guidance := "slightly richer vocabulary, but still clear and child-safe"
	if age <= 5 {
		guidance = "very simple vocabulary, tiny sentences, one clear action"
	} else if age <= 7 {
		guidance = "simple sentences, familiar vocabulary, light cause-and-effect"
	}
	return fmt.Sprintf("CRITICAL: You are generating for a %d year old child. Use the %s age tier: story_text MUST be %d-%d words and MUST NOT exceed %d words. Use %s. NO complex paragraphs.", age, tier, minWords, maxWords, maxWords, guidance)
}

func CountWords(text string) int {
	return len(strings.Fields(stripHTMLTags(text)))
}

func IndonesianMarkers(text string) []string {
	lower := strings.ToLower(stripHTMLTags(text))
	phrases := []string{
		"apa kabar", "rumah sakit", "uang saku", "nggak mau", "gak mau",
		"ngapain kamu", "gue mau", "lu mau", "bego banget", "jelek sekali",
	}
	var markers []string
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			markers = append(markers, phrase)
		}
	}

	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicodeLetterOrNumber(r)
	})
	tokenMarkers := map[string]string{
		"gak":     "gak/nggak",
		"nggak":   "gak/nggak",
		"banget":  "banget",
		"gue":     "gue",
		"lu":      "lu",
		"ngapain": "ngapain",
		"mau":     "mau",
		"uang":    "uang",
		"sepeda":  "sepeda",
		"bego":    "bego",
		"jelek":   "jelek",
		"dong":    "dong",
		"sih":     "sih",
		"ngerti":  "ngerti",
		"bikin":   "bikin",
		"ngerasa": "ngerasa",
		"nulis":   "nulis",
		"bilang":  "bilang",
		"aja":     "aja",
		"udah":    "udah",
		"belom":   "belom",
		"kalo":    "kalo",
		"emang":   "emang",
		"kayak":   "kayak",
		"gini":    "gini",
		"gitu":    "gitu",
		"trus":    "trus",
		"dimana":  "dimana",
		"gimana":  "gimana",
		"pake":    "pake",
		"gede":    "gede",
		"dapet":   "dapet",
		"nyari":   "nyari",
		"liat":    "liat",
		"denger":  "denger",
		"bener":   "bener",
		"nyokap":  "nyokap",
		"bokap":   "bokap",
		"halu":    "halu",
		"lebay":   "lebay",
		"cupu":    "cupu",
		"norak":   "norak",
		"alay":    "alay",
		"ngasih":  "ngasih",
		"makasih": "makasih",
		"karna":   "karna",
		"nyampe":  "nyampe",
		"nyobain": "nyobain",
		"taruh":   "taruh",
		"deket":   "deket",
		"cobain":  "cobain",
		"matiin":  "matiin",
		"bagusin": "bagusin",
		"rusak":   "rusak",
	}
	seen := make(map[string]bool)
	for _, field := range fields {
		label, ok := tokenMarkers[field]
		if !ok || seen[label] {
			continue
		}
		seen[label] = true
		markers = append(markers, label)
	}
	return markers
}

// indonesianReplacements and SanitizeBahasaMalaysia have been removed.
// Indonesian detection is now handled by the LLM self-review pass
// (Engine.reviewBahasaMalaysia). IndonesianMarkers is kept for logging.

func domainKidsAgeTier(age int) string {
	if age <= 5 {
		return "4-5"
	}
	if age <= 7 {
		return "6-7"
	}
	return "8+"
}

func kidsWordBounds(age int) (int, int) {
	if age <= 5 {
		return 1, 20
	}
	if age <= 7 {
		return 20, 40
	}
	return 40, 80
}

func stripHTMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func unicodeLetterOrNumber(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '\''
}

func KidsMaxWords(age int) int {
	_, maxWords := kidsWordBounds(age)
	return maxWords
}

func EnforceKidsWordLimit(text string, age int) string {
	maxWords := KidsMaxWords(age)
	if maxWords <= 0 {
		return text
	}
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}
	return strings.Join(words[:maxWords], " ")
}

func BuildKidsImagePrompt(imageScene string, entities interface{}, genre string, age int) string {
	genre = cleanPromptPart(genre, 40)
	characters := kidsCharacterVisuals(entities)
	scene := cleanPromptPart(imageScene, 200)
	if scene == "" {
		scene = "a bright child-safe storybook scene with the characters present"
	}
	if characters == "" {
		characters = "consistent child protagonist with clear hair, clothing, shoes, and accessory details"
	}
	return fmt.Sprintf("Malaysian children's storybook illustration, age %d, warm cheerful watercolor style, no text, no scary imagery. Characters: %s. Scene: %s.", age, characters, scene)
}

func kidsCharacterVisuals(entities interface{}) string {
	var values []domain.Entity
	switch typed := entities.(type) {
	case map[string]domain.Entity:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			values = append(values, typed[key])
		}
	case []domain.Entity:
		values = typed
	}
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, min(len(values), 3))
	for _, ent := range values {
		if ent.Name == "" {
			continue
		}
		appearance := cleanPromptPart(ent.Appearance, 140)
		if appearance == "" && len(ent.Traits) > 0 {
			appearance = cleanPromptPart(strings.Join(ent.Traits, ", "), 100)
		}
		role := cleanPromptPart(ent.Role, 40)
		if role == "" {
			role = cleanPromptPart(ent.RelationToPC, 40)
		}
		name := cleanPromptPart(ent.Name, 40)
		if appearance != "" {
			parts = append(parts, strings.TrimSpace(fmt.Sprintf("%s (%s), %s", name, role, appearance)))
		} else {
			parts = append(parts, strings.TrimSpace(fmt.Sprintf("%s (%s)", name, role)))
		}
		if len(parts) >= 3 {
			break
		}
	}
	return strings.Join(parts, "; ")
}

func kidsSceneAction(storyText string) string {
	clean := scriptBlock.ReplaceAllString(storyText, "")
	clean = stripHTMLTags(clean)
	clean = strings.Join(strings.Fields(clean), " ")
	clean = promptUnsafeChars.ReplaceAllString(clean, "")
	if clean == "" {
		return ""
	}
	sentences := sentenceSplitter.Split(clean, -1)
	parts := make([]string, 0, 2)
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		parts = append(parts, sentence)
		if len(parts) == 2 {
			break
		}
	}
	return cleanPromptPart(strings.Join(parts, ". "), 140)
}

func cleanPromptPart(text string, maxLen int) string {
	clean := scriptBlock.ReplaceAllString(text, "")
	clean = stripHTMLTags(clean)
	clean = strings.Join(strings.Fields(clean), " ")
	clean = promptUnsafeChars.ReplaceAllString(clean, "")
	clean = strings.TrimSpace(clean)
	if maxLen > 0 && len(clean) > maxLen {
		clean = strings.TrimSpace(clean[:maxLen])
	}
	return clean
}

func KidsImageURL(imageScene string, entities interface{}, genre string, age int) string {
	prompt := BuildKidsImagePrompt(imageScene, entities, genre, age)
	return "/api/kids/image?prompt=" + url.QueryEscape(prompt)
}

var scriptBlock = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)

var sentenceSplitter = regexp.MustCompile(`[.!?]+\s*`)

var promptUnsafeChars = regexp.MustCompile(`[^\p{L}\p{N}\s,.'-]`)
