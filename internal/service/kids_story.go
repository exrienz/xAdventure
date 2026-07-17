package service

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/muz/xadventure/internal/domain"
)

const (
	KidsStoryMinPages  = 6
	KidsStoryPageCount = 10
)

func KidsStyleRules(genre string, age int, dynamicKidsEnabled bool) string {
	minWords, maxWords := kidsWordBounds(age)
	targetWords := kidsTargetWords(age, 1)
	pagePlan := kidsArcOverview()
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
- Every page must feel like the next natural step. Never start like the story is already halfway through a crisis.
- Cover only ONE clear micro-beat per page so the story grows gradually and smoothly.
- Establish exactly ONE clear central problem, need, or goal for the story.
- After that central problem appears, keep the whole story focused on solving that SAME problem.
- Do NOT replace the main problem with a new unrelated mission, mystery, villain, object, or side quest.
- Obstacles may delay the solution, but they must make the SAME main problem harder rather than starting a different story.
- Every choice must be a sensible next action toward understanding, handling, or solving the SAME main problem.
- Do NOT give random, silly, bait, or unrelated choices that drag the story away from its core issue.
- The story must last at least 6 pages and at most 10 pages.
- End each non-final page with a simple, age-appropriate hook or question.
- If the story reaches a full, satisfying ending on page 6, 7, 8, or 9, you may finish naturally there.
- On the final story page, set story_complete to true in the JSON.
- Use this page arc as guidance:
` + pagePlan
	if !dynamicKidsEnabled {
		return prompt
	}

	return fmt.Sprintf(`%s

Dynamic Age-Based Page Length (Age %d):
- HARD LIMIT: each page's story_text MUST be %d-%d words. DO NOT EXCEED %d words.
- Aim for about %d words on each page unless a slightly shorter or longer page serves the story better.
- %s
- Use %s
- Choices must also follow the same language, tone, and simplicity rules.`, prompt, normalizedKidsAge(age), minWords, maxWords, maxWords, targetWords, kidsWordGuidance(age), kidsSentenceGuidance(age))
}

func KidsDynamicAgeInstruction(age int) string {
	minWords, maxWords := kidsWordBounds(age)
	return fmt.Sprintf("CRITICAL: You are generating one page for a %d year old child. Keep story_text between %d-%d words and NEVER exceed %d words. %s Use %s. Cover only one small story beat on this page.", normalizedKidsAge(age), minWords, maxWords, maxWords, kidsWordGuidance(age), kidsSentenceGuidance(age))
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

func kidsWordBounds(age int) (int, int) {
	switch normalizedKidsAge(age) {
	case 3:
		return 10, 20
	case 4:
		return 15, 30
	case 5:
		return 25, 50
	case 6:
		return 40, 80
	case 7:
		return 70, 120
	default:
		return 100, 180
	}
}

func normalizedKidsAge(age int) int {
	if age < 3 {
		return 3
	}
	if age > 8 {
		return 8
	}
	return age
}

func kidsWordGuidance(age int) string {
	switch normalizedKidsAge(age) {
	case 3:
		return "Use very tiny vocabulary with one action and one clear feeling"
	case 4:
		return "Use very simple vocabulary with tiny actions and obvious cause-and-effect"
	case 5:
		return "Use simple familiar vocabulary with a gentle beginning, action, and tiny hook"
	case 6:
		return "Use simple sentences with light cause-and-effect and one meaningful change"
	case 7:
		return "Use clear school-age vocabulary with steady progression and one focused obstacle"
	default:
		return "Use richer but still child-safe vocabulary with smoother detail, motivation, and payoff"
	}
}

func kidsSentenceGuidance(age int) string {
	switch normalizedKidsAge(age) {
	case 3:
		return "2-3 ultra-short sentences"
	case 4:
		return "2-4 very short sentences"
	case 5:
		return "3-5 short sentences"
	case 6:
		return "4-6 short sentences"
	case 7:
		return "5-7 clear sentences"
	default:
		return "6-9 clear sentences"
	}
}

func kidsArcOverview() string {
	steps := []string{
		"Page 1: gentle opening, protagonist, place, and small want",
		"Page 2: a clue, invitation, or small problem appears",
		"Page 3: the protagonist takes the first real step",
		"Page 4: an early obstacle or surprise slows them down",
		"Page 5: they learn, find, or realize something useful",
		"Page 6: a setback or mistake makes the same main problem harder; from here the story may end naturally once that main problem is fully solved",
		"Page 7: the protagonist tries again with more courage or teamwork",
		"Page 8: the biggest child-safe obstacle appears before the ending",
		"Page 9: the solution begins and the main problem starts to unlock",
		"Page 10: if the story is still active, this must be the complete happy ending, calm landing, and clear moral lesson",
	}
	return "- " + strings.Join(steps, "\n- ")
}

func KidsPageInstruction(turnNumber, age int) string {
	minWords, maxWords := kidsWordBounds(age)
	targetWords := kidsTargetWords(age, turnNumber)
	common := fmt.Sprintf("PAGE %d OF %d. Keep story_text between %d-%d words and never exceed %d words. Aim for about %d words. Use %s. %s. Cover only one clear micro-beat on this page. Budget the page so the final 4-8 words can finish a complete closing sentence or hook. Never stop mid-sentence, mid-thought, or on a dangling fragment. Do NOT rush ahead to later pages.", turnNumber, KidsStoryPageCount, minWords, maxWords, maxWords, targetWords, kidsWordGuidance(age), kidsSentenceGuidance(age))
	switch turnNumber {
	case 1:
		return common + " Start at the true beginning. Introduce the protagonist, the place, and one small want or activity. Do NOT begin in the middle of danger, chaos, or a half-finished adventure. End with a tiny clue or invitation."
	case 2:
		return common + " Show the first clue, invitation, or gentle problem. Let the protagonist notice it and care about it. Make this the same main problem the whole story will solve."
	case 3:
		return common + " Show the protagonist taking the first real step toward the same main problem. Keep the action small and easy to follow."
	case 4:
		return common + " Add an early obstacle, misunderstanding, or playful surprise. The problem should grow naturally from the earlier pages. Do NOT introduce a different main problem."
	case 5:
		return common + " Give the protagonist a useful discovery, helper, tool, or lesson that helps with the same main problem. This is the middle of the journey, not the ending."
	case 6:
		return common + " Show a meaningful obstacle, turning point, or breakthrough connected to the same main problem. If the story may end naturally here because one brave, kind, or clever action can fully solve the main problem, do it now, end warmly, and set story_complete to true. Do NOT save the ending for page 10 just because more pages are available. Otherwise use this page to prepare the final stretch without starting a new problem."
	case 7:
		return common + " Let the protagonist try again with more bravery, kindness, or teamwork on the same main problem. If the main problem becomes fully solved in a satisfying way here, end the story now and set story_complete to true. Do NOT hold back a finished ending for later pages. Otherwise keep moving toward the ending."
	case 8:
		return common + " Present a strong child-safe obstacle or turning point before the ending. It must still belong to the same main problem. If the story naturally resolves here after that turning point, end fully, add a warm landing, and set story_complete to true. Do NOT stretch a solved story to page 10. Otherwise keep building toward the ending."
	case 9:
		return common + " Begin or complete the solution to the same main problem. If the main problem is fully resolved here, set story_complete to true and end warmly without leaving anything hanging. Do NOT end this page with only a clue, realization, or plan if the solution can already happen now."
	case 10:
		return common + " This is the final page. Fully solve the same main problem on this page, show the solution actually happening, give a happy and satisfying ending, add a calm landing, state a gentle moral lesson, and set story_complete to true. Do NOT end with only a realization, clue, or future plan. Do NOT ask a question at the end."
	default:
		return common + " Keep the story smooth, child-safe, connected to the previous page, and focused on the same main problem."
	}
}

func KidsPageIndicator(turnNumber int) string {
	if turnNumber < 1 {
		turnNumber = 1
	}
	if turnNumber > KidsStoryPageCount {
		turnNumber = KidsStoryPageCount
	}
	return "Halaman " + strconv.Itoa(turnNumber) + " / maks " + strconv.Itoa(KidsStoryPageCount)
}

func kidsTargetWords(age, turnNumber int) int {
	minWords, maxWords := kidsWordBounds(age)
	if maxWords <= minWords {
		return maxWords
	}
	rangeWords := maxWords - minWords
	target := minWords + (rangeWords * 3 / 4)
	if turnNumber <= 2 {
		target = minWords + (rangeWords * 2 / 3)
	} else if turnNumber >= 9 {
		target = minWords + (rangeWords * 4 / 5)
	}
	if target > maxWords-2 {
		target = maxWords - 2
	}
	if target < minWords {
		target = minWords
	}
	return target
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
	text = strings.TrimSpace(text)
	maxWords := KidsMaxWords(age)
	if maxWords <= 0 {
		return text
	}
	minWords, _ := kidsWordBounds(age)
	words := strings.Fields(text)
	if len(words) > maxWords {
		text = trimToWordBudget(text, maxWords)
	}
	text = trimIncompleteTail(text, minWords)
	if !endsWithTerminalPunctuation(text) {
		text = strings.TrimSpace(trimDanglingWords(text))
		if text != "" && !endsWithTerminalPunctuation(text) {
			text += "."
		}
	}
	return strings.TrimSpace(text)
}

func trimToWordBudget(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return strings.TrimSpace(text)
	}
	truncated := strings.Join(words[:maxWords], " ")
	if idx := lastSentenceBoundary(truncated); idx >= 0 {
		candidate := strings.TrimSpace(truncated[:idx+1])
		if candidate != "" {
			return candidate
		}
	}
	return strings.TrimSpace(trimDanglingWords(truncated))
}

func trimIncompleteTail(text string, minWords int) string {
	text = strings.TrimSpace(text)
	if text == "" || endsWithTerminalPunctuation(text) {
		return text
	}
	if idx := lastSentenceBoundary(text); idx >= 0 {
		candidate := strings.TrimSpace(text[:idx+1])
		if CountWords(candidate) >= minWords {
			return candidate
		}
	}
	return text
}

func lastSentenceBoundary(text string) int {
	last := -1
	for i, r := range text {
		switch r {
		case '.', '!', '?':
			last = i
		}
	}
	return last
}

func endsWithTerminalPunctuation(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	runes := []rune(text)
	last := runes[len(runes)-1]
	return last == '.' || last == '!' || last == '?'
}

func trimDanglingWords(text string) string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return ""
	}
	dangling := map[string]bool{
		"dan": true, "atau": true, "yang": true, "di": true, "ke": true, "dari": true,
		"dengan": true, "untuk": true, "kerana": true, "supaya": true, "sambil": true,
		"lalu": true, "semasa": true, "ketika": true, "bila": true, "agar": true,
		"dia": true, "itu": true, "ini": true, "mereka": true, "kami": true, "kita": true,
		"saya": true, "kamu": true, "nya": true,
	}
	for len(words) > 1 {
		last := strings.ToLower(strings.Trim(words[len(words)-1], `"'.,!?`))
		if !dangling[last] {
			break
		}
		words = words[:len(words)-1]
	}
	return strings.Join(words, " ")
}

func BuildKidsImagePrompt(imageScene, visualSetting string, entities interface{}, genre string, age int) string {
	genre = cleanPromptPart(genre, 40)
	characters := kidsCharacterVisuals(entities)
	setting := cleanPromptPart(visualSetting, 180)
	scene := kidsSingleMomentScene(imageScene)
	if scene == "" {
		scene = "a bright child-safe moment with the characters present"
	}
	if characters == "" {
		characters = "consistent child protagonist with clear hair, clothing, shoes, and accessory details"
	}
	if setting == "" {
		setting = fmt.Sprintf("consistent %s environment with stable background details and a child-safe atmosphere", strings.ToLower(genre))
	}
	return fmt.Sprintf("A single full-bleed children's watercolor illustration. One camera angle. One continuous environment. One uninterrupted composition. Everything exists in one physical space. Depict one freeze-frame moment in time. No split composition or multiple scenes. Editorial children's illustration, warm Malaysian watercolor style, age %d, no text, no letters, no numbers, no scary imagery. Maintain consistent character designs and environment. Characters: %s. Setting: %s. Scene: %s.", age, characters, setting, scene)
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

func kidsSingleMomentScene(imageScene string) string {
	clean := cleanPromptPart(imageScene, 220)
	if clean == "" {
		return ""
	}
	sentences := sentenceSplitter.Split(clean, -1)
	first := ""
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		first = sentence
		break
	}
	if first == "" {
		first = clean
	}
	lower := strings.ToLower(first)
	for _, marker := range []string{
		" then ", " after that ", " next ", " suddenly ", " finally ", " later ",
	} {
		if idx := strings.Index(lower, marker); idx > 0 {
			first = strings.TrimSpace(first[:idx])
			break
		}
	}
	return cleanPromptPart(first, 140)
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

func KidsImageURL(imageScene, visualSetting string, entities interface{}, genre string, age int) string {
	prompt := BuildKidsImagePrompt(imageScene, visualSetting, entities, genre, age)
	return "/api/kids/image?prompt=" + url.QueryEscape(prompt)
}

var scriptBlock = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)

var sentenceSplitter = regexp.MustCompile(`[.!?]+\s*`)

var promptUnsafeChars = regexp.MustCompile(`[^\p{L}\p{N}\s,.'-]`)
