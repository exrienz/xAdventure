package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math/big"
	"strings"

	"github.com/muz/xadventure/internal/domain"
)

// TwistEngine decides when to inject plot twists into the narrative.
type TwistEngine struct {
	wildness int // 0-100, higher means more twists
}

func NewTwistEngine(wildness int) *TwistEngine {
	if wildness < 0 {
		wildness = 0
	}
	if wildness > 100 {
		wildness = 100
	}
	return &TwistEngine{wildness: wildness}
}

// RollTwist returns a twist level and a flavor text instruction.
// The LLM is also free to add its own twist, but this gives it permission.
func (te *TwistEngine) RollTwist(turnNumber int, twistsUsed int, genre string) (domain.TwistLevel, string) {
	// Base probabilities
	minorChance := 15
	majorChance := 5
	wildChance := 2

	// Increase chances slightly as story progresses
	progressBonus := turnNumber / 10
	minorChance += progressBonus
	majorChance += progressBonus / 2
	wildChance += progressBonus / 4

	// Wildness setting scales everything up
	minorChance += te.wildness / 3
	majorChance += te.wildness / 6
	wildChance += te.wildness / 10

	roll, _ := rand.Int(rand.Reader, big.NewInt(100))
	v := int(roll.Int64())

	if v < wildChance {
		return domain.TwistWild, te.wildTwistInstruction(genre)
	}
	if v < wildChance+majorChance {
		return domain.TwistMajor, te.majorTwistInstruction(genre)
	}
	if v < wildChance+majorChance+minorChance {
		return domain.TwistMinor, te.minorTwistInstruction(genre)
	}
	return domain.TwistNone, ""
}

func (te *TwistEngine) minorTwistInstruction(genre string) string {
	options := []string{
		"Introduce an unexpected minor complication: a sudden noise, a shift in weather, a strange symbol appearing, or an NPC reacting oddly.",
		"Reveal a small but unsettling detail that changes how the protagonist interprets the current situation.",
		"Add a moment of dramatic irony: the reader (and maybe the protagonist) realizes something is off.",
		"Introduce a fleeting NPC, object, or omen that may become important later.",
		"Describe a subtle but meaningful change in a companion's behavior or the environment.",
	}
	if !isDarkGenre(genre) {
		options = []string{
			"Introduce an unexpected minor complication: a sudden change in weather, a lost item, or a miscommunication.",
			"Reveal a small detail that makes the protagonist reconsider their current plan.",
			"Add a moment of mild irony: the reader realizes a misunderstanding is occurring.",
			"Introduce a fleeting NPC or object that hints at a larger secret without being threatening.",
			"Describe a subtle but meaningful change in a companion's mood or the environment.",
		}
	}
	return te.pick(options)
}

func (te *TwistEngine) majorTwistInstruction(genre string) string {
	options := []string{
		"Reveal that someone is not who they appear to be, or that the protagonist has been lied to.",
		"Introduce a sudden betrayal, ambush, or reversal of fortune.",
		"Shift the setting dramatically: a fire, collapse, teleportation, or arrival of a powerful force.",
		"Reveal a hidden connection between two characters or factions.",
		"Force the protagonist into a morally difficult decision with no clean answer.",
	}
	if !isDarkGenre(genre) {
		options = []string{
			"Reveal a hidden truth about a character's past that completely changes the dynamic.",
			"Introduce a sudden complication: an unexpected rival appears or a critical opportunity is lost.",
			"Shift the setting dramatically: an abrupt change of location or a sudden important event.",
			"Reveal a surprising connection between two characters or factions.",
			"Force the protagonist into a difficult decision between two strongly desired outcomes.",
		}
	}
	return te.pick(options)
}

func (te *TwistEngine) wildTwistInstruction(genre string) string {
	options := []string{
		"Drop a massive revelation that recontextualizes the entire scene or quest.",
		"Introduce an impossible event: time loops, a dying character speaking prophecy, reality glitching.",
		"Bring back an old choice as a devastating consequence.",
		"Make the protagonist lose something they thought was safe, or gain something deeply cursed.",
		"End the scene with a cliffhanger no one saw coming.",
	}
	if !isDarkGenre(genre) {
		options = []string{
			"Drop a massive revelation that completely recontextualizes the protagonist's goals or relationships.",
			"Introduce an incredibly unlikely but dramatically appropriate coincidence.",
			"Bring back an old choice as a surprisingly major consequence.",
			"Make the protagonist risk losing something very important to them.",
			"End the scene with a shocking and emotional cliffhanger.",
		}
	}
	return te.pick(options)
}

func isDarkGenre(genre string) bool {
	g := strings.ToLower(genre)
	return g == "horror" || g == "post-apocalyptic" || g == "cyberpunk" || g == "noir" || g == "mystery"
}

func (te *TwistEngine) pick(options []string) string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(options))))
	return options[n.Int64()]
}

// GenerateSeed creates a random seed string for story replayability.
func GenerateSeed() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// TwistFlavor returns a human-readable hint of what kind of twist happened.
func TwistFlavor(level domain.TwistLevel) string {
	switch level {
	case domain.TwistMinor:
		return "A subtle tension rises..."
	case domain.TwistMajor:
		return "The story takes a sharp turn!"
	case domain.TwistWild:
		return "Everything changes."
	default:
		return ""
	}
}

func ArchetypeDescription(arch string) string {
	switch strings.ToLower(arch) {
	case "hero":
		return "brave, self-sacrificing, drawn to protect others"
	case "betrayer":
		return "cunning, willing to break trust, always calculating"
	case "trickster":
		return "witty, unpredictable, turns problems into jokes"
	case "survivor":
		return "cautious, resourceful, prioritizes staying alive"
	case "seeker":
		return "curious, obsessed with secrets, risks everything for truth"
	case "caretaker":
		return "empathetic, heals and protects, driven by bonds"
	case "outcast":
		return "marked by rejection, sharp-edged, hard to trust"
	case "chosen":
		return "burdened by destiny, reluctant but powerful"
	default:
		return "resourceful and adaptable"
	}
}

func GenreDescription(genre string) string {
	switch strings.ToLower(genre) {
	case "adventure", "fantasy adventure":
		return "A world of sword, spell, ancient ruins, and forgotten gods."
	case "romance":
		return "A story of hearts, longing, and the cost of connection."
	case "cyberpunk":
		return "Neon streets, megacorps, hackers, and synthetic souls."
	case "horror":
		return "Dread, survival, and things that should not exist."
	case "sci-fi":
		return "Starships, alien worlds, and the edge of human knowledge."
	case "mystery":
		return "Secrets, suspects, and the slow unraveling of truth."
	case "post-apocalyptic":
		return "Ash, ruin, and the fragile hope of rebuilding."
	case "supernatural":
		return "Ghosts, curses, and the veil between worlds."
	case "steampunk":
		return "Brass gears, airships, and clockwork revolution."
	case "xianxia":
		return "Cultivation, immortal sects, and the path to transcendence."
	case "isekai":
		return "Transported to another world, fate rewritten from zero."
	case "noir":
		return "Rain-slick streets, femme fatales, and moral shadows."
	// Kids genres
	case "pengembaraan":
		return "Pengembaraan menarik, pencarian harta karun, dan penerokaan dunia baru yang selamat untuk kanak-kanak."
	case "fantasi":
		return "Dunia ajaib dengan sihir, makhluk mitologi, dan tempat-tempat imaginasi yang menakjubkan."
	case "kisah dongeng":
		return "Dongeng tradisional dengan raja, ratu, sihir, dan pengajaran yang bermakna untuk kanak-kanak."
	case "cerita haiwan":
		return "Pengembaraan haiwan yang menunjukkan sifat manusia, persahabatan, dan kebaikan hati."
	case "persahabatan":
		return "Tema persahabatan, kerja berpasukan, empati, dan kebaikan hati antara kawan-kawan."
	case "keluarga":
		return "Cerita tentang adik-beradik, ibu bapa, datuk nenek, dan hubungan kekeluargaan yang hangat."
	case "kelakar":
		return "Pengembaraan lucu, situasi jenaka, dan watak-watak yang kelakar untuk kanak-kanak."
	case "mistik/misteri":
		return "Misteri sesuai umur dengan petunjuk, unsur mistik yang lembut, dan suspen yang menyeronokkan untuk kanak-kanak."
	case "sains & teknologi":
		return "Cerita yang memperkenalkan konsep sains, teknologi, kejuruteraan, dan matematik secara semula jadi."
	case "inspirasi":
		return "Cerita yang menggalakkan ketabahan, kebaikan, kesyukuran, keberanian, dan ketekunan."
	default:
		return "A tale of danger and discovery."
	}
}

func BuildOpening(sceneType string) string {
	switch sceneType {
	case "calm":
		return "Start with a quiet, normal scene. Then slowly make it feel strange or dangerous."
	case "action":
		return "Start in the middle of action. Then explain how the character got there."
	case "mystery":
		return "Start with the character finding something that should not be there."
	case "dialogue":
		return "Start with a tense talk between two people. Show that something bigger is wrong."
	default:
		return "Start with a clear, interesting opening scene."
	}
}

func OpeningScene() string {
	scenes := []string{"calm", "action", "mystery", "dialogue"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(scenes))))
	return scenes[n.Int64()]
}

func KidsSettingSpark(genre, seed string) string {
	profile, ok := kidsSettingProfiles[strings.ToLower(strings.TrimSpace(genre))]
	if !ok {
		profile = kidsSettingProfile{
			places:   []string{"a lively neighborhood corner", "a quiet riverside path", "a bright village square"},
			features: []string{"fluttering paper lanterns", "a small hidden object", "a surprising sound in the air"},
			actions:  []string{"someone notices a tiny mystery", "a simple task turns unusual", "a new clue quietly appears"},
			moods:    []string{"curious and hopeful", "playful but mysterious", "gentle and inviting"},
		}
	}

	rng := newSeededRNG(kidsSettingSeed(genre, seed))
	place := rng.pickString(profile.places)
	feature := rng.pickString(profile.features)
	action := rng.pickString(profile.actions)
	mood := rng.pickString(profile.moods)

	return fmt.Sprintf("%s, with %s, where %s. Mood: %s.", place, feature, action, mood)
}

type kidsSettingProfile struct {
	places   []string
	features []string
	actions  []string
	moods    []string
}

var kidsSettingProfiles = map[string]kidsSettingProfile{
	"pengembaraan": {
		places: []string{
			"a hidden bamboo footpath near a rushing stream",
			"an old hilltop watchpost overlooking a windy valley",
			"a narrow trail behind a fruit orchard and a broken fence",
			"a tiny jetty beside a slow green river",
			"a forgotten path between rice fields and low stone markers",
		},
		features: []string{
			"a folded map tucked in a bottle",
			"fresh footprints beside a mossy sign",
			"a wooden chest key hanging from a ribbon",
			"a bright compass that trembles softly",
			"a distant flag moving between the trees",
		},
		actions: []string{
			"a small clue invites the hero to explore farther",
			"a normal walk suddenly feels like the start of a quest",
			"an errand becomes the first step of a journey",
			"a tiny discovery points toward a bigger adventure",
			"the hero notices a path no one mentioned before",
		},
		moods: []string{"excited and curious", "fresh and adventurous", "hopeful and brave", "playful and daring"},
	},
	"fantasi": {
		places: []string{
			"a lantern-lit garden where leaves shimmer like tiny stars",
			"a moon-pale clearing beside a whispering crystal pond",
			"a cloud-soft meadow behind a silver gate",
			"a floating bridge over glowing mist",
			"a secret grove where flowers hum in the breeze",
		},
		features: []string{
			"a pebble glowing in rainbow colors",
			"a sleeping mushroom ring with silver dust",
			"a tiny winged creature hiding in the petals",
			"a ribbon of light circling an old stump",
			"a bell-shaped flower opening by itself",
		},
		actions: []string{
			"magic peeks into an ordinary afternoon",
			"the hero notices a strange wonder no one else has seen",
			"a small fantasy clue asks to be followed",
			"the world feels kind but quietly enchanted",
			"something impossible appears in a gentle way",
		},
		moods: []string{"warm and magical", "gentle and sparkling", "dreamy and safe", "wonder-filled and bright"},
	},
	"cerita haiwan": {
		places: []string{
			"a sunny burrow village under a hibiscus hedge",
			"a shady pond path lined with cattails",
			"a cozy treetop nook above a sleepy lane",
			"a grassy field beside a duck-filled stream",
			"a small barn corner with warm hay and birdsong",
		},
		features: []string{
			"a tiny scarf caught on a twig",
			"a basket of berries tipped on the ground",
			"a trail of shiny seeds",
			"a wobbling acorn cart",
			"a small bell ringing from behind the reeds",
		},
		actions: []string{
			"an animal friend needs a little help",
			"the animals notice a problem that only teamwork can solve",
			"a playful misunderstanding starts the story",
			"a gentle mystery appears among the creatures",
			"a kind animal spots something unusual nearby",
		},
		moods: []string{"cozy and friendly", "gentle and playful", "caring and curious", "warm and cheerful"},
	},
	"kisah dongeng": {
		places: []string{
			"a small kingdom road beside a golden wheat field",
			"a cottage edge near a moonlit well",
			"a castle garden behind an ivy-covered wall",
			"a winding lane leading to a tall old tower",
			"a forest path beside a ribbon-blue brook",
		},
		features: []string{
			"a silver key wrapped in cloth",
			"a talking bird feather on the path",
			"a locked box no one remembers",
			"a spinning wheel thread caught on a rose bush",
			"a tiny crown charm hidden in the grass",
		},
		actions: []string{
			"a simple day begins to feel like a true fairy tale",
			"the hero discovers a gentle sign of old magic",
			"a mysterious object hints at a classic quest",
			"someone kind is called toward a royal secret",
			"an old tale suddenly feels real",
		},
		moods: []string{"timeless and magical", "soft and wondrous", "classic and hopeful", "gentle and enchanted"},
	},
	"keluarga": {
		places: []string{
			"a busy kitchen filled with afternoon light",
			"a wooden veranda beside potted plants and slippers",
			"a family courtyard after a light rain",
			"a living room with folded blankets and old photo frames",
			"a backyard where clothes sway on a line",
		},
		features: []string{
			"an old tin box under a chair",
			"a missing recipe page",
			"a small parcel tied with string",
			"a framed photo turned face down",
			"a familiar song humming from another room",
		},
		actions: []string{
			"a family errand turns meaningful",
			"a small household problem brings everyone together",
			"the hero notices something important about home",
			"an ordinary family moment opens a gentle mystery",
			"a simple discovery reveals a warm memory",
		},
		moods: []string{"warm and homely", "loving and gentle", "nostalgic and safe", "soft and caring"},
	},
	"kelakar": {
		places: []string{
			"a school canteen corner during a breezy break",
			"a backyard with buckets, slippers, and wobbling stools",
			"a market lane full of funny sounds",
			"a playground edge near a squeaky swing",
			"a kitchen where something keeps clattering",
		},
		features: []string{
			"a hat stuck in the wrong place",
			"a very bouncy parcel",
			"a squeaking toy no one can catch",
			"a tray that slides a little too far",
			"a suspiciously wiggly basket",
		},
		actions: []string{
			"a silly misunderstanding starts a chain of laughs",
			"something ordinary goes comically wrong",
			"the hero notices a funny problem that keeps growing",
			"a tiny mistake turns into playful chaos",
			"everyone tries to stay serious, but cannot",
		},
		moods: []string{"light and goofy", "playful and funny", "cheerful and silly", "bouncy and bright"},
	},
	"persahabatan": {
		places: []string{
			"a school field after the morning bell",
			"a shaded bench near a mural wall",
			"a library reading corner with soft sunlight",
			"a small bridge in the park between two paths",
			"a craft table beside open classroom windows",
		},
		features: []string{
			"a shared notebook left behind",
			"a bracelet bead rolled under a bench",
			"a half-finished drawing with two names on it",
			"a friendship ribbon caught on a bag zipper",
			"a paper star folded by a classmate",
		},
		actions: []string{
			"two friends notice a problem they can solve together",
			"a small misunderstanding asks for kindness",
			"a quiet moment becomes the start of teamwork",
			"one child realizes a friend needs help",
			"a shared discovery deepens a friendship",
		},
		moods: []string{"warm and trusting", "gentle and cooperative", "kind and uplifting", "friendly and hopeful"},
	},
	"inspirasi": {
		places: []string{
			"a sunrise field beside a small village track",
			"a simple classroom before the others arrive",
			"a workshop corner with tools and paper plans",
			"a community garden beside a narrow path",
			"a quiet practice space under a broad tree",
		},
		features: []string{
			"a list of goals with one blank space",
			"a small broken item waiting to be repaired",
			"a handmade badge tucked in a pocket",
			"a seedling that needs extra care",
			"a note of encouragement from someone older",
		},
		actions: []string{
			"the hero faces a challenge that asks for patience",
			"a small chance to do the right thing appears",
			"effort and kindness begin in an ordinary moment",
			"the day offers a chance to be brave in a quiet way",
			"a simple responsibility turns meaningful",
		},
		moods: []string{"hopeful and uplifting", "steady and brave", "gentle and motivating", "bright and encouraging"},
	},
	"sains & teknologi": {
		places: []string{
			"a tidy science corner with jars and labeled boxes",
			"a school lab table near an open window",
			"a small robot workshop in the back room",
			"a rooftop lookout with a homemade weather station",
			"a garden path beside a row of curious gadgets",
		},
		features: []string{
			"a blinking sensor with one loose wire",
			"a notebook full of neat sketches",
			"a tiny solar toy that suddenly stops",
			"a model bridge waiting to be tested",
			"a curious magnet pulling the wrong object",
		},
		actions: []string{
			"a question leads to a hands-on discovery",
			"the hero notices something that does not behave as expected",
			"a simple experiment begins with a surprise",
			"a small invention asks to be understood",
			"a problem invites careful thinking and testing",
		},
		moods: []string{"curious and clever", "bright and inventive", "playful and thoughtful", "focused and exciting"},
	},
	"mistik/misteri": {
		places: []string{
			"a quiet veranda at dusk beside a darkening yard",
			"a narrow lane near an old locked storeroom",
			"a misty riverside path under swaying bamboo",
			"a sleepy village corner after the evening call to prayer",
			"a dim garden path where lantern light barely reaches",
		},
		features: []string{
			"a smooth stone wrapped in faded cloth",
			"a shadow moving where no one stands",
			"a note with half the words missing",
			"a small charm hanging from a nail",
			"a strange sound coming from behind a wooden wall",
		},
		actions: []string{
			"a quiet mystery appears in an ordinary place",
			"the hero notices something that should not be there",
			"an elder's warning turns a normal evening strange",
			"a tiny clue makes the air feel different",
			"something familiar suddenly carries a secret",
		},
		moods: []string{"calm but mysterious", "gentle and suspenseful", "quiet and curious", "softly eerie but safe"},
	},
}

func kidsSettingSeed(genre, seed string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(genre)) + "::" + strings.TrimSpace(seed)))
	return int64(h.Sum64() & 0x7fffffffffffffff)
}

// FormatAsLightNovel returns any extra prompt text that forces light-novel style.
func FormatAsLightNovel(genre string) string {
	if domain.IsKidsGenre(genre) {
		return `Style Rules (Strict Bahasa Malaysia for Kids):
- You MUST write the ENTIRE story text and choices in standard Bahasa Malaysia only.
- Use standard Malaysian Malay vocabulary and spelling. Do NOT use Indonesian dialect, slang, or syntax.
- STRICTLY FORBIDDEN Indonesian words (use Malaysian equivalent): gak/nggak→tidak, banget→sangat,
  gue→saya, lu→kamu, ngapain→buat apa, mau→mahu, uang→wang, sepeda→basikal, apa kabar→apa khabar,
  rumah sakit→hospital, bego→bodoh, jelek→buruk, dong/sih→(omit), ngerti→faham, bikin→buat,
  bilang→kata, aja→saja, udah→sudah, belom→belum, kalo→kalau, emang→memang, kayak→macam,
  gini→begini, gitu→begitu, trus→terus, dimana→di mana, gimana→bagaimana, pake→pakai,
  gede→besar, dapet→dapat, nyari→cari, liat→lihat, denger→dengar, bener→betul.
- Use Malaysian vocabulary: cikgu, murid, tandas, kereta api, basikal, kemeja, setem, tiket,
  kerusi, meja, almari, katil, bantal, selimut, tingkap, peti sejuk, mesin basuh, lampu, suis,
  motosikal, telefon, televisyen, wayar, plag, mentol, kasut, stoking, seluar, songkok, tudung.
- The output MUST be 100% Bahasa Malaysia. Mixing with Bahasa Indonesia is NEVER acceptable.
- If you catch yourself using an Indonesian word, replace it with the Malaysian equivalent immediately.
- Use extremely simple, clear, and very short sentences.
- MUST be engaging, playful, and fun! Include onomatopoeia (e.g., "Bum!", "Meow!", "Wush!") and exciting interactions to keep kids entertained.
- Do NOT use dark, scary, violent, romantic, or overly dramatic themes.
- The tone must be friendly, encouraging, and playful.
- Write in third-person limited point of view.
- Every page must feel like the next natural step. Never start like the story is already halfway through a crisis.
- End each non-final page with a simple, engaging question or hook.
- STORY ARC: The story must last at least 6 pages and at most 10 pages, with a gentle opening, gradual build, and a natural happy ending.
- If the main problem is fully and satisfyingly resolved on page 6, 7, 8, or 9, you may end there naturally. Otherwise page 10 must be the final ending.
- Each page MUST advance the story meaningfully. Do NOT stall, skip ahead suddenly, or repeat.`
	}

	return `Style Rules (Simple English Light Novel):
- Use short, clear sentences. Avoid fancy or rare words.
- Write in third-person limited point of view, showing the main character's thoughts.
- Use plain dialogue tags like "he said" or "she asked".
- Include at least one spoken line each turn if another person or creature is there.
- End every turn on a small cliffhanger, feeling, or surprise.
- Use common words. Keep the reading easy and fast.
- Mark internal thoughts with *asterisks* around the thought text.`
}

// FormatHintText gives a small UI flavor for the current turn.
func FormatHintText(level domain.TwistLevel, turn int) string {
	if level != domain.TwistNone {
		return fmt.Sprintf("%s (%s)", TwistFlavor(level), strings.ToUpper(string(level)))
	}
	if turn%6 == 0 {
		return "End of Chapter"
	}
	return ""
}
