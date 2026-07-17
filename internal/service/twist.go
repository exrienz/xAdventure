package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
