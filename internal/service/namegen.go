package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/muz/xadventure/internal/domain"
)

// NameGenerator produces genre-appropriate character names while avoiding
// repetition across recent generations.
type NameGenerator struct {
	mu        sync.Mutex
	history   []string
	capacity  int
	safety    *SafetyFilter
	usedInRun map[string]bool
}

// NewNameGenerator creates a generator that remembers the last `capacity` names.
func NewNameGenerator(capacity int, safety *SafetyFilter) *NameGenerator {
	if capacity <= 0 {
		capacity = 50
	}
	return &NameGenerator{
		history:   make([]string, 0, capacity),
		capacity:  capacity,
		safety:    safety,
		usedInRun: make(map[string]bool),
	}
}

// GeneratedName is the result of a name generation attempt.
type GeneratedName struct {
	Name     string
	Seed     string
	IsUnique bool
}

// Generate returns a genre-appropriate name that is not in the recent history.
// It will make several attempts before falling back to a fresh neutral profile.
func (ng *NameGenerator) Generate(genre string) GeneratedName {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	seed := time.Now().UnixNano() + int64(randInt(1<<20))
	rng := newSeededRNG(seed)

	for attempt := 0; attempt < 12; attempt++ {
		name := ng.pickForGenre(genre, rng)
		if name == "" {
			continue
		}
		if ng.safety != nil && !ng.safety.IsSafeName(name) {
			continue
		}
		if !ng.recentlyUsed(name) && !ng.usedInRun[strings.ToLower(name)] {
			ng.record(name)
			ng.usedInRun[strings.ToLower(name)] = true
			return GeneratedName{Name: name, Seed: fmt.Sprintf("%d", seed+int64(attempt)), IsUnique: true}
		}
	}

	for attempt := 0; attempt < 6; attempt++ {
		name := normalizeGeneratedName(buildProceduralName(neutralNameProfile(), rng))
		if name == "" {
			continue
		}
		if ng.safety != nil && !ng.safety.IsSafeName(name) {
			continue
		}
		ng.record(name)
		ng.usedInRun[strings.ToLower(name)] = true
		return GeneratedName{Name: name, Seed: fmt.Sprintf("%d", seed+100+int64(attempt)), IsUnique: false}
	}

	// The generator always has fragments to build from, so reaching here should
	// be extremely rare. Keep retrying procedurally rather than falling back to a fixed label.
	for attempt := 0; attempt < 8; attempt++ {
		name := normalizeGeneratedName(buildProceduralName(neutralNameProfile(), rng))
		if name == "" {
			continue
		}
		ng.record(name)
		ng.usedInRun[strings.ToLower(name)] = true
		return GeneratedName{Name: name, Seed: fmt.Sprintf("%d", seed+999+int64(attempt)), IsUnique: false}
	}

	return GeneratedName{Name: "", Seed: fmt.Sprintf("%d", seed+1999), IsUnique: false}
}

// Reset clears the per-request deduplication cache. Keep history intact.
func (ng *NameGenerator) Reset() {
	ng.mu.Lock()
	defer ng.mu.Unlock()
	ng.usedInRun = make(map[string]bool)
}

// UniqueRate returns the percentage of names in history that are not in the
// provided slice of recent names. A high value means the generator is producing
// fresh names.
func (ng *NameGenerator) UniqueRate(recent []string) float64 {
	ng.mu.Lock()
	defer ng.mu.Unlock()
	if len(ng.history) == 0 {
		return 1.0
	}
	recentSet := make(map[string]bool, len(recent))
	for _, n := range recent {
		recentSet[strings.ToLower(strings.TrimSpace(n))] = true
	}
	unique := 0
	for _, n := range ng.history {
		if !recentSet[strings.ToLower(strings.TrimSpace(n))] {
			unique++
		}
	}
	return float64(unique) / float64(len(ng.history))
}

// History returns a snapshot of recently generated names.
func (ng *NameGenerator) History() []string {
	ng.mu.Lock()
	defer ng.mu.Unlock()
	out := make([]string, len(ng.history))
	copy(out, ng.history)
	return out
}

func (ng *NameGenerator) pickForGenre(genre string, rng *seededRNG) string {
	return normalizeGeneratedName(buildProceduralName(profileForGenre(genre), rng))
}

func (ng *NameGenerator) recentlyUsed(name string) bool {
	key := strings.ToLower(strings.TrimSpace(name))
	for _, h := range ng.history {
		if strings.ToLower(strings.TrimSpace(h)) == key {
			return true
		}
	}
	return false
}

func (ng *NameGenerator) record(name string) {
	if len(ng.history) >= ng.capacity {
		ng.history = ng.history[1:]
	}
	ng.history = append(ng.history, name)
}

type segmentProfile struct {
	onsets          []string
	vowels          []string
	codas           []string
	prefixes        []string
	suffixes        []string
	minSyllables    int
	maxSyllables    int
	openVowelChance int
	midCodaChance   int
	prefixChance    int
	suffixChance    int
}

type nameFormat struct {
	parts  []segmentProfile
	joiner string
}

type nameProfile struct {
	formats []nameFormat
}

func profileForGenre(genre string) nameProfile {
	lower := strings.ToLower(strings.TrimSpace(genre))
	switch lower {
	case "romance":
		return lyricalNameProfile()
	case "cyberpunk":
		return cyberpunkNameProfile()
	case "horror":
		return gothicNameProfile()
	case "sci-fi":
		return astralNameProfile()
	case "mystery":
		return sleuthNameProfile()
	case "post-apocalyptic":
		return wastelandNameProfile()
	case "supernatural":
		return etherealNameProfile()
	case "steampunk":
		return brassNameProfile()
	case "xianxia":
		return jadeNameProfile()
	case "isekai":
		return brightKanaNameProfile()
	case "noir":
		return noirNameProfile()
	case "adventure", "fantasy adventure":
		return mythicNameProfile()
	}

	if domain.IsKidsGenre(genre) {
		switch lower {
		case "fantasi", "dongeng klasik", "budaya dan folklor", "mistik":
			return kidsWonderNameProfile()
		case "sains dan teknologi", "fiksyen sains kanak-kanak", "edukasi", "inspirasi":
			return kidsDiscoveryNameProfile()
		default:
			return kidsWarmNameProfile()
		}
	}

	return neutralNameProfile()
}

func buildProceduralName(profile nameProfile, rng *seededRNG) string {
	if len(profile.formats) == 0 {
		profile = neutralNameProfile()
	}
	format := profile.formats[rng.intn(len(profile.formats))]
	parts := make([]string, 0, len(format.parts))
	for _, spec := range format.parts {
		part := buildSegment(spec, rng)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, format.joiner)
}

func buildSegment(spec segmentProfile, rng *seededRNG) string {
	if spec.minSyllables <= 0 {
		spec.minSyllables = 1
	}
	if spec.maxSyllables < spec.minSyllables {
		spec.maxSyllables = spec.minSyllables
	}
	count := spec.minSyllables
	if spec.maxSyllables > spec.minSyllables {
		count += rng.intn(spec.maxSyllables - spec.minSyllables + 1)
	}

	var b strings.Builder
	if len(spec.prefixes) > 0 && rng.intn(100) < spec.prefixChance {
		b.WriteString(rng.pickString(spec.prefixes))
	}

	for i := 0; i < count; i++ {
		onset := ""
		if len(spec.onsets) > 0 && !(i == 0 && rng.intn(100) < spec.openVowelChance) {
			onset = rng.pickString(spec.onsets)
		}
		vowel := rng.pickString(spec.vowels)
		coda := ""
		if len(spec.codas) > 0 {
			if i == count-1 || rng.intn(100) < spec.midCodaChance {
				coda = rng.pickString(spec.codas)
			}
		}
		b.WriteString(onset)
		b.WriteString(vowel)
		b.WriteString(coda)
	}

	if len(spec.suffixes) > 0 && rng.intn(100) < spec.suffixChance {
		b.WriteString(rng.pickString(spec.suffixes))
	}

	return smoothGeneratedPart(b.String())
}

func smoothGeneratedPart(raw string) string {
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	var b strings.Builder
	var prev rune
	repeats := 0
	for i, r := range lower {
		if i > 0 && r == prev {
			repeats++
			if repeats >= 2 {
				continue
			}
		} else {
			repeats = 0
		}
		b.WriteRune(r)
		prev = r
	}
	out := b.String()
	replacer := strings.NewReplacer(
		"'''", "'",
		"--", "-",
		"aae", "ae",
		"eea", "ea",
		"iio", "io",
		"uua", "ua",
	)
	return replacer.Replace(out)
}

func normalizeGeneratedName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.Fields(raw)
	for i, part := range parts {
		parts[i] = titleCaseToken(part)
	}
	return strings.Join(parts, " ")
}

func titleCaseToken(token string) string {
	pieces := strings.Split(token, "-")
	for i, piece := range pieces {
		pieces[i] = capitalize(piece)
	}
	return strings.Join(pieces, "-")
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func mythicNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"b", "br", "c", "cl", "d", "dr", "f", "g", "gl", "k", "kr", "l", "m", "n", "r", "s", "st", "t", "th", "v"},
		vowels:          []string{"a", "e", "i", "o", "u", "ae", "ia", "io", "oa"},
		codas:           []string{"", "n", "r", "s", "l", "m", "th", "nd", "ric", "ra"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 12,
		midCodaChance:   20,
	}
	family := segmentProfile{
		onsets:          []string{"ash", "black", "bright", "dawn", "dr", "fair", "fell", "iron", "moon", "moss", "north", "raven", "storm", "thorn", "vale", "wind", "wyr"},
		vowels:          []string{"a", "e", "i", "o", "u", "oa", "ea"},
		codas:           []string{"", "n", "r", "s", "d", "th", "wood", "hart", "born", "veil"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 5,
		midCodaChance:   30,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
	}}
}

func lyricalNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"b", "c", "cl", "d", "f", "h", "j", "l", "m", "n", "r", "s", "t", "v"},
		vowels:          []string{"a", "e", "i", "o", "u", "ia", "ea", "io", "ai"},
		codas:           []string{"", "n", "l", "r", "s", "m", "na", "ra", "elle", "ine"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 20,
		midCodaChance:   15,
	}
	family := segmentProfile{
		onsets:          []string{"bell", "car", "del", "ever", "fair", "laur", "mor", "ros", "val", "whit"},
		vowels:          []string{"a", "e", "i", "o", "u", "ea", "io"},
		codas:           []string{"", "n", "l", "r", "tte", "line", "vale", "mont"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 10,
		midCodaChance:   15,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
	}}
}

func cyberpunkNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"c", "cr", "d", "dr", "j", "k", "kr", "n", "q", "r", "s", "sk", "sy", "t", "tr", "v", "x", "z"},
		vowels:          []string{"a", "e", "i", "o", "u", "y", "ae", "io"},
		codas:           []string{"", "k", "n", "x", "z", "v", "r", "s", "th", "q"},
		prefixes:        []string{"neo", "syn", "vox"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 8,
		midCodaChance:   35,
		prefixChance:    18,
	}
	family := segmentProfile{
		onsets:          []string{"byte", "cry", "grid", "hex", "ion", "kry", "nex", "pulse", "rax", "volt", "wire", "zen"},
		vowels:          []string{"a", "e", "i", "o", "u", "y"},
		codas:           []string{"", "n", "r", "x", "v", "k", "sh"},
		suffixes:        []string{"-ix", "-01", "-vx"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 0,
		midCodaChance:   30,
		suffixChance:    15,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: "-"},
	}}
}

func gothicNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"b", "bl", "c", "cr", "d", "g", "gr", "l", "m", "n", "r", "s", "v", "w"},
		vowels:          []string{"a", "e", "i", "o", "u", "ae", "io"},
		codas:           []string{"", "n", "r", "s", "l", "m", "th", "d", "mour", "vane"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 10,
		midCodaChance:   25,
	}
	family := segmentProfile{
		onsets:          []string{"black", "crow", "dread", "grave", "hallow", "mourn", "night", "pale", "rav", "wither"},
		vowels:          []string{"a", "e", "i", "o", "u", "oa"},
		codas:           []string{"", "n", "r", "th", "wood", "mere", "fall", "croft"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 5,
		midCodaChance:   25,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
	}}
}

func astralNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"a", "c", "el", "f", "h", "k", "l", "m", "n", "r", "s", "t", "v", "x", "z"},
		vowels:          []string{"a", "e", "i", "o", "u", "ae", "ia", "io", "oa", "ui"},
		codas:           []string{"", "n", "r", "s", "l", "x", "th", "ra", "ron"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 18,
		midCodaChance:   18,
	}
	family := segmentProfile{
		onsets:          []string{"astro", "cel", "nova", "orbit", "quasar", "sol", "stell", "vega", "zen"},
		vowels:          []string{"a", "e", "i", "o", "u", "io", "oa"},
		codas:           []string{"", "n", "r", "s", "x", "is", "ion"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 10,
		midCodaChance:   18,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
	}}
}

func sleuthNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"a", "b", "c", "d", "f", "g", "h", "l", "m", "p", "r", "s", "t", "v", "w"},
		vowels:          []string{"a", "e", "i", "o", "u", "ai", "ea"},
		codas:           []string{"", "n", "r", "s", "l", "m", "d", "tt", "son"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 12,
		midCodaChance:   22,
	}
	family := segmentProfile{
		onsets:          []string{"ash", "black", "bram", "croft", "frost", "mar", "quill", "ster", "thorn", "wren"},
		vowels:          []string{"a", "e", "i", "o", "u", "oa"},
		codas:           []string{"", "n", "r", "s", "ford", "well", "ham", "croft"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 5,
		midCodaChance:   25,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
	}}
}

func wastelandNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"b", "c", "d", "f", "g", "k", "m", "n", "r", "s", "t", "w", "z"},
		vowels:          []string{"a", "e", "i", "o", "u", "ae"},
		codas:           []string{"", "k", "n", "r", "x", "sh", "sk", "m", "t"},
		prefixes:        []string{"dust", "rust"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 5,
		midCodaChance:   35,
		prefixChance:    14,
	}
	family := segmentProfile{
		onsets:          []string{"ash", "bar", "cairn", "dune", "flint", "grim", "iron", "scar", "stone", "thorn"},
		vowels:          []string{"a", "e", "i", "o", "u"},
		codas:           []string{"", "n", "r", "k", "d", "fall", "mark", "wind"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 5,
		midCodaChance:   28,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: "-"},
	}}
}

func etherealNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"a", "el", "f", "h", "l", "m", "n", "r", "s", "th", "v", "y"},
		vowels:          []string{"a", "e", "i", "o", "u", "ae", "ea", "ia", "io", "ui"},
		codas:           []string{"", "n", "r", "l", "s", "th", "riel", "vyn"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 25,
		midCodaChance:   12,
	}
	family := segmentProfile{
		onsets:          []string{"moon", "night", "raven", "shade", "silver", "star", "veil", "whisper", "wisp"},
		vowels:          []string{"a", "e", "i", "o", "u", "ia"},
		codas:           []string{"", "n", "r", "l", "fall", "mere", "bloom"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 10,
		midCodaChance:   15,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
	}}
}

func brassNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"al", "bar", "cl", "ed", "f", "g", "h", "m", "p", "r", "t", "v"},
		vowels:          []string{"a", "e", "i", "o", "u", "ea", "io"},
		codas:           []string{"", "n", "r", "l", "s", "d", "ric", "bert", "ton"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 10,
		midCodaChance:   22,
	}
	family := segmentProfile{
		onsets:          []string{"brass", "cog", "copper", "gear", "iron", "penny", "steam", "tin", "whit"},
		vowels:          []string{"a", "e", "i", "o", "u"},
		codas:           []string{"", "n", "r", "l", "ford", "well", "wright", "croft"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 5,
		midCodaChance:   24,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
	}}
}

func jadeNameProfile() nameProfile {
	family := segmentProfile{
		onsets:          []string{"b", "c", "f", "h", "j", "l", "m", "q", "s", "w", "x", "y", "z", "zh"},
		vowels:          []string{"a", "e", "i", "o", "u", "ia", "iu", "ao"},
		codas:           []string{"", "n", "ng"},
		minSyllables:    1,
		maxSyllables:    1,
		openVowelChance: 5,
		midCodaChance:   0,
	}
	given := segmentProfile{
		onsets:          []string{"b", "ch", "d", "f", "g", "h", "j", "l", "m", "q", "r", "s", "t", "w", "x", "y", "z", "zh"},
		vowels:          []string{"a", "e", "i", "o", "u", "ai", "ao", "ia", "ie", "iu", "uo"},
		codas:           []string{"", "n", "ng"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 3,
		midCodaChance:   5,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{family, given}, joiner: " "},
		{parts: []segmentProfile{family, given}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
	}}
}

func brightKanaNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"k", "s", "t", "n", "h", "m", "r", "y", "w", "sh", "ch", "ts"},
		vowels:          []string{"a", "e", "i", "o", "u", "ya", "yo", "yu"},
		codas:           []string{"", "n"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 4,
		midCodaChance:   0,
	}
	family := segmentProfile{
		onsets:          []string{"a", "i", "k", "m", "n", "s", "t", "y", "h", "f", "w"},
		vowels:          []string{"a", "e", "i", "o", "u", "ai", "ei", "ou"},
		codas:           []string{"", "n"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 10,
		midCodaChance:   0,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{family, given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
	}}
}

func noirNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"b", "c", "d", "f", "g", "j", "l", "m", "r", "s", "t", "v"},
		vowels:          []string{"a", "e", "i", "o", "u", "ea"},
		codas:           []string{"", "n", "r", "s", "l", "m", "ck", "tt", "d"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 8,
		midCodaChance:   25,
	}
	family := segmentProfile{
		onsets:          []string{"bard", "car", "del", "finn", "gray", "marl", "more", "sull", "val", "voss"},
		vowels:          []string{"a", "e", "i", "o", "u"},
		codas:           []string{"", "n", "r", "s", "o", "an", "ett", "ero"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 4,
		midCodaChance:   20,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
	}}
}

func kidsWarmNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"b", "c", "d", "g", "h", "j", "k", "l", "m", "n", "p", "r", "s", "t", "w", "y"},
		vowels:          []string{"a", "e", "i", "o", "u", "ai", "ia"},
		codas:           []string{"", "n", "m", "l", "r"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 10,
		midCodaChance:   8,
	}
	family := segmentProfile{
		onsets:          []string{"ba", "da", "ka", "la", "ma", "na", "ra", "sa", "ta"},
		vowels:          []string{"a", "e", "i", "o", "u"},
		codas:           []string{"", "n", "m", "h"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 15,
		midCodaChance:   5,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
	}}
}

func kidsWonderNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"b", "c", "d", "f", "g", "h", "k", "l", "m", "n", "p", "r", "s", "t", "w", "y"},
		vowels:          []string{"a", "e", "i", "o", "u", "ia", "io", "ai"},
		codas:           []string{"", "n", "l", "r", "m"},
		suffixes:        []string{"ya", "ri", "na"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 15,
		midCodaChance:   6,
		suffixChance:    14,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
	}}
}

func kidsDiscoveryNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"a", "b", "c", "d", "f", "g", "k", "l", "m", "n", "p", "r", "s", "t", "v", "z"},
		vowels:          []string{"a", "e", "i", "o", "u", "ia", "io"},
		codas:           []string{"", "n", "m", "r", "s"},
		prefixes:        []string{"di", "ki", "ri"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 18,
		midCodaChance:   10,
		prefixChance:    12,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, given}, joiner: " "},
	}}
}

func neutralNameProfile() nameProfile {
	given := segmentProfile{
		onsets:          []string{"b", "c", "d", "f", "g", "h", "j", "k", "l", "m", "n", "p", "r", "s", "t", "v", "w", "y", "z"},
		vowels:          []string{"a", "e", "i", "o", "u", "ai", "ia", "io"},
		codas:           []string{"", "n", "r", "s", "l", "m"},
		minSyllables:    2,
		maxSyllables:    3,
		openVowelChance: 15,
		midCodaChance:   12,
	}
	family := segmentProfile{
		onsets:          []string{"b", "c", "d", "f", "g", "k", "l", "m", "n", "r", "s", "t", "v", "w"},
		vowels:          []string{"a", "e", "i", "o", "u", "ia"},
		codas:           []string{"", "n", "r", "s", "l", "m", "d"},
		minSyllables:    1,
		maxSyllables:    2,
		openVowelChance: 10,
		midCodaChance:   12,
	}
	return nameProfile{formats: []nameFormat{
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given}, joiner: " "},
		{parts: []segmentProfile{given, family}, joiner: " "},
	}}
}

// seededRNG is a tiny deterministic PRNG for name generation.
// It does not need to be cryptographically secure; it only needs
// reproducibility for the metrics seed.
type seededRNG struct {
	seed int64
}

func newSeededRNG(seed int64) *seededRNG {
	return &seededRNG{seed: seed}
}

func (r *seededRNG) intn(n int) int {
	if n <= 0 {
		return 0
	}
	r.seed = (r.seed*1103515245 + 12345) & 0x7fffffffffffffff
	return int(r.seed % int64(n))
}

func (r *seededRNG) pickString(opts []string) string {
	if len(opts) == 0 {
		return ""
	}
	return opts[r.intn(len(opts))]
}

func randInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if n == nil {
		return 0
	}
	return int(n.Int64())
}

// IsValidNameRequest validates a user-supplied name and rejects anything that
// looks like a prompt-injection payload.
func IsValidNameRequest(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	if len(name) > 80 {
		return false
	}
	lower := strings.ToLower(name)
	bad := []string{"ignore", "disregard", "system prompt", "above instruction", "jailbreak", "<|", "```", "{{", "}}", "http://", "https://"}
	for _, b := range bad {
		if strings.Contains(lower, b) {
			return false
		}
	}
	return true
}

// FallbackName returns a procedurally generated name when the user-supplied
// name is rejected.
func FallbackName(genre string) string {
	seed := time.Now().UnixNano() + int64(randInt(1<<20))
	rng := newSeededRNG(seed)
	profile := profileForGenre(genre)
	safety := NewSafetyFilter()
	for attempt := 0; attempt < 8; attempt++ {
		name := normalizeGeneratedName(buildProceduralName(profile, rng))
		if name != "" && safety.IsSafeName(name) {
			return name
		}
	}
	for attempt := 0; attempt < 8; attempt++ {
		name := normalizeGeneratedName(buildProceduralName(neutralNameProfile(), rng))
		if name != "" && safety.IsSafeName(name) {
			return name
		}
	}
	return normalizeGeneratedName(buildProceduralName(kidsWarmNameProfile(), rng))
}

// NewUniqueNameGeneratorForTests exposes a generator seeded with a fixed seed
// for deterministic tests. Production code should use NewNameGenerator.
func NewUniqueNameGeneratorForTests(capacity int, safety *SafetyFilter) *NameGenerator {
	return NewNameGenerator(capacity, safety)
}
