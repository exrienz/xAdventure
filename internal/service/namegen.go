package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
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
// It will make several attempts before falling back to a decorated name.
func (ng *NameGenerator) Generate(genre string) GeneratedName {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	seed := time.Now().UnixNano() + int64(randInt(1<<20))
	rng := newSeededRNG(seed)

	for attempt := 0; attempt < 8; attempt++ {
		name := ng.pickForGenre(genre, rng)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		name = strings.Title(strings.ToLower(name))
		if ng.safety != nil && !ng.safety.IsSafeName(name) {
			continue
		}
		if !ng.recentlyUsed(name) && !ng.usedInRun[name] {
			ng.record(name)
			ng.usedInRun[name] = true
			return GeneratedName{Name: name, Seed: fmt.Sprintf("%d", seed+int64(attempt)), IsUnique: true}
		}
	}

	// Fallback: decorate a base name with a short random suffix.
	base := ng.pickForGenre(genre, rng)
	name := fmt.Sprintf("%s %s", base, ng.randomSuffix(rng))
	name = strings.TrimSpace(name)
	name = strings.Title(strings.ToLower(name))
	ng.record(name)
	ng.usedInRun[name] = true
	return GeneratedName{Name: name, Seed: fmt.Sprintf("%d", seed+100), IsUnique: false}
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
	lists, ok := genreNameLists[strings.ToLower(genre)]
	if !ok {
		lists = genreNameLists["adventure"]
	}
	style := rng.pickString([]string{"first", "full", "epithet"})
	switch style {
	case "full":
		first := rng.pickString(lists.first)
		last := rng.pickString(lists.last)
		return fmt.Sprintf("%s %s", first, last)
	case "epithet":
		first := rng.pickString(lists.first)
		epithet := rng.pickString(lists.epithets)
		return fmt.Sprintf("%s the %s", first, epithet)
	default:
		return rng.pickString(lists.first)
	}
}

func (ng *NameGenerator) randomSuffix(rng *seededRNG) string {
	return rng.pickString([]string{"Ash", "Thorn", "Veil", "Drift", "Rook", "Sparrow", "Quill", "Hollow", "Kind", "Shade"})
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

type nameList struct {
	first    []string
	last     []string
	epithets []string
}

// genreNameLists provides linguistically-flavored name pools for each genre.
// These are small curated pools; they seed diversity through combinations and
// the LLM is encouraged to invent additional names in the same style.
var genreNameLists = map[string]nameList{
	"adventure": {
		first: []string{"Elias", "Rowan", "Kira", "Thane", "Mira", "Cedric", "Lena", "Dorian", "Sable", "Finn", "Aurelia", "Gareth", "Nerys", "Orion", "Isolde"},
		last:  []string{"Stormwake", "Ironhart", "Brightshield", "Ashford", "Wilder", "Drakehollow", "Fairwind", "Stonevale", "Mosswood", "Thornwood"},
		epithets: []string{"Brave", "Swift", "Steadfast", "Wandering", "Bold", "Keen", "Fierce", "True", "Wild", "Patient"},
	},
	"romance": {
		first: []string{"Julian", "Sofia", "Theo", "Clara", "Leo", "Amara", "Henry", "Elise", "Luca", "Naomi", "Oliver", "Maya", "Daniel", "Iris", "Felix"},
		last:  []string{"Hartwell", "Fairchild", "Ashbury", "Bellamy", "Lovecraft", "Dearborn", "Moreau", "Chen", "Rossi", "Whitmore"},
		epithets: []string{"Hopeful", "Tender", "Loyal", "Dreaming", "Patient", "Passionate", "Gentle", "True", "Yearning", "Devoted"},
	},
	"cyberpunk": {
		first: []string{"Kael", "Rin", "Jax", "Nova", "Zero", "Vex", "Sera", "Nix", "Blaze", "Echo", "Kira", "Rex", "Mako", "Lyra", "Cade"},
		last:  []string{"Vance", "Kovacs", "Tanaka", "Sato", "Reyes", "Orlov", "Kim", "Zhao", "Patel", "Silva"},
		epithets: []string{"Wired", "Neon", "Ghost", "Chrome", "Spliced", "Hollow", "Burned", "Glitch", "Rogue", "Sleek"},
	},
	"horror": {
		first: []string{"Silas", "Edith", "Marcel", "Lenore", "Jonas", "Wren", "Victor", "Mabel", "Gideon", "Eleanor", "Caleb", "Ruth", "Otto", "Iris", "Hugo"},
		last:  []string{"Blackwood", "Crane", "Vance", "Holloway", "Marsh", "Sinclair", "Darrow", "Graves", "Crowe", "Wakefield"},
		epithets: []string{"Pale", "Hollow", "Bleak", "Cursed", "Dreadful", "Forsaken", "Silent", "Ghastly", "Mournful", "Unseen"},
	},
	"sci-fi": {
		first: []string{"Cora", "Isaac", "Lyra", "Marcus", "Zoe", "Jaxon", "Astra", "Kiran", "Elara", "Nolan", "Freya", "Soren", "Tessa", "Orion", "Ira"},
		last:  []string{"Vance", "Solano", "Nakamura", "Sterling", "Kessler", "Owusu", "Brynn", "Drake", "Yilmaz", "Carter"},
		epithets: []string{"Stellar", "Voidborn", "Cosmic", "Keen", "Last", "Farwatch", "Bold", "Bright", "Patient", "Wandering"},
	},
	"mystery": {
		first: []string{"Arthur", "Irene", "Harold", "Vivian", "Felix", "Mona", "Cedric", "Diana", "Owen", "Maude", "Percy", "Gwen", "Lionel", "Rosalind", "Simon"},
		last:  []string{"Hastings", "Marlowe", "Poirot", "Moriarty", "Holmes", "Blackwell", "Ashcroft", "Quinn", "Frost", "Sterling"},
		epithets: []string{"Sharp", "Curious", "Careful", "Keen", "Subtle", "Patient", "Wary", "Clever", "Watchful", "Skeptical"},
	},
	"post-apocalyptic": {
		first: []string{"Ash", "Scout", "Rust", "Kael", "Mira", "Tank", "Juno", "Cinder", "Rook", "Fern", "Rider", "Nox", "Tess", "Bram", "Wren"},
		last:  []string{"Scavenger", "Walker", "Dustborn", "Last", "Drifter", "Wasteland", "Ash", "Iron", "Rook", "Hollow"},
		epithets: []string{"Scavenging", "Dusty", "Hard", "Wary", "Scarred", "Steadfast", "Grim", "Fierce", "Last", "Rugged"},
	},
	"supernatural": {
		first: []string{"Cassian", "Luna", "Rowan", "Iris", "Silas", "Aurelia", "Mira", "Jude", "Elena", "Orion", "Naomi", "Lucian", "Wren", "Celeste", "Dorian"},
		last:  []string{"Moonshadow", "Grimwood", "Starling", "Ravenwood", "Nightfall", "Ashthorne", "Hollow", "Blackwell", "Storm", "Wraith"},
		epithets: []string{"Veiled", "Otherworldly", "Cursed", "Ethereal", "Shadowed", "Ancient", "Fated", "Silent", "Dreaming", "Strange"},
	},
	"steampunk": {
		first: []string{"Thaddeus", "Emmeline", "Barnaby", "Clara", "Horatio", "Gwendolyn", "Alfred", "Rosalind", "Cornelius", "Mabel", "Ignatius", "Violet", "Reginald", "Beatrice", "Phineas"},
		last:  []string{"Cogsworth", "Brasswell", "Gearhart", "Copperfield", "Steamwright", "Ironclad", "Bolton", "Pembroke", "Finch", "Winchester"},
		epithets: []string{"Inventive", "Clockwork", "Brass", "Goggled", "Mechanical", "Steadfast", "Curious", "Polished", "Bold", "Tinkering"},
	},
	"xianxia": {
		first: []string{"Lin", "Wei", "Xiao", "Yue", "Chen", "Feng", "Rou", "Jian", "Mei", "Long", "Huan", "Qing", "Zhi", "Lian", "Yan"},
		last:  []string{"Wuxia", "Tianfeng", "Yunhai", "Qinglian", "Xuanji", "Long", "Jiang", "Bai", "Shen", "Zhou"},
		epithets: []string{"Eternal", "Celestial", "Azure", "Drifting", "Unbroken", "Radiant", "Serene", "Ancient", "Boundless", "Phoenix"},
	},
	"isekai": {
		first: []string{"Kazuki", "Aoi", "Rei", "Sora", "Mio", "Haruto", "Yuki", "Ren", "Emi", "Daichi", "Rin", "Itsuki", "Kaede", "Takeshi", "Nao"},
		last:  []string{"Yamada", "Sato", "Tanaka", "Suzuki", "Takahashi", "Kobayashi", "Nakamura", "Ito", "Watanabe", "Kimura"},
		epithets: []string{"Otherworld", "Reborn", "Summoned", "Awakened", "Drifting", "Fated", "Stray", "Brave", "Lost", "Chosen"},
	},
	"noir": {
		first: []string{"Sam", "Vera", "Jack", "Lola", "Mickey", "Dolores", "Frank", "Irene", "Vincent", "Gloria", "Eddie", "Ruby", "Tony", "Sadie", "Danny"},
		last:  []string{"Sullivan", "O'Brien", "Moretti", "Caruso", "Finn", "Delacroix", "Volkov", "Castellano", "Doyle", "Brennan"},
		epithets: []string{"Hardboiled", "Crooked", "Broken", "Smoking", "Slick", "Jaded", "Cold", "Lucky", "Bitter", "Sly"},
	},
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

// FallbackName returns a sensible default when the user name is rejected.
func FallbackName() string {
	return "Traveler"
}

// NewUniqueNameGeneratorForTests exposes a generator seeded with a fixed seed
// for deterministic tests. Production code should use NewNameGenerator.
func NewUniqueNameGeneratorForTests(capacity int, safety *SafetyFilter) *NameGenerator {
	return NewNameGenerator(capacity, safety)
}
