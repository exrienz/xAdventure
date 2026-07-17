package domain

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidChoice = errors.New("invalid choice")

// ValidGenres is the allowlist of supported genres.
var ValidGenres = []string{
	// Adult genres
	"Adventure", "Romance", "Cyberpunk", "Horror", "Sci-Fi", "Mystery",
	"Post-Apocalyptic", "Supernatural", "Steampunk", "Xianxia", "Isekai", "Noir",
	// Kids genres
	"Kids",
	"Pengembaraan", "Fantasi", "Dongeng Klasik", "Fabel", "Cerita Haiwan",
	"Cerita Sebelum Tidur", "Edukasi", "Persahabatan", "Keluarga", "Humor",
	"Misteri Kanak-kanak", "Fiksyen Sains Kanak-kanak", "Fiksyen Sejarah",
	"Alam dan Persekitaran", "Membesar", "Budaya dan Folklor", "Mistik",
	"Cerita Interaktif", "Sains dan Teknologi", "Inspirasi",
}

// ValidArchetypes is the allowlist of supported archetypes.
var ValidArchetypes = []string{
	"Hero", "Betrayer", "Trickster", "Survivor", "Seeker", "Caretaker", "Outcast", "Chosen",
}

// GenreArchetypeMatrix defines which archetypes are allowed in which genres.
var GenreArchetypeMatrix = map[string][]string{
	"Adventure":        {"Hero", "Betrayer", "Trickster", "Survivor", "Seeker", "Caretaker", "Outcast", "Chosen"},
	"Romance":          {"Hero", "Trickster", "Caretaker", "Outcast", "Seeker"},
	"Cyberpunk":        {"Betrayer", "Trickster", "Survivor", "Seeker", "Outcast", "Hero"},
	"Horror":           {"Survivor", "Seeker", "Caretaker", "Outcast", "Betrayer"},
	"Sci-Fi":           {"Hero", "Betrayer", "Trickster", "Survivor", "Seeker", "Caretaker", "Outcast", "Chosen"},
	"Mystery":          {"Seeker", "Trickster", "Betrayer", "Outcast", "Survivor", "Caretaker"},
	"Post-Apocalyptic": {"Survivor", "Betrayer", "Caretaker", "Outcast", "Hero", "Trickster"},
	"Supernatural":     {"Seeker", "Chosen", "Survivor", "Outcast", "Trickster", "Betrayer"},
	"Steampunk":        {"Trickster", "Seeker", "Hero", "Outcast", "Caretaker", "Betrayer"},
	"Xianxia":          {"Chosen", "Hero", "Seeker", "Betrayer", "Outcast", "Trickster"},
	"Isekai":           {"Chosen", "Hero", "Trickster", "Caretaker", "Survivor", "Outcast"},
	"Noir":             {"Seeker", "Survivor", "Betrayer", "Outcast", "Trickster"},
	// Kids genres — all support the same safe subset.
	"Kids":                      {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Pengembaraan":              {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Fantasi":                   {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Dongeng Klasik":            {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Fabel":                     {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Cerita Haiwan":             {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Cerita Sebelum Tidur":      {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Edukasi":                   {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Persahabatan":              {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Keluarga":                  {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Humor":                     {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Misteri Kanak-kanak":       {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Fiksyen Sains Kanak-kanak": {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Fiksyen Sejarah":           {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Alam dan Persekitaran":     {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Membesar":                  {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Budaya dan Folklor":        {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Mistik":                    {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Cerita Interaktif":         {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Sains dan Teknologi":       {"Hero", "Trickster", "Caretaker", "Seeker"},
	"Inspirasi":                 {"Hero", "Trickster", "Caretaker", "Seeker"},
}

func IsValidGenre(g string) bool {
	for _, v := range ValidGenres {
		if v == g {
			return true
		}
	}
	return false
}

func IsKidsGenre(g string) bool {
	switch strings.ToLower(g) {
	case "kids",
		"pengembaraan", "fantasi", "dongeng klasik", "fabel", "cerita haiwan",
		"cerita sebelum tidur", "edukasi", "persahabatan", "keluarga", "humor",
		"misteri kanak-kanak", "fiksyen sains kanak-kanak", "fiksyen sejarah",
		"alam dan persekitaran", "membesar", "budaya dan folklor", "mistik",
		"cerita interaktif", "sains dan teknologi", "inspirasi":
		return true
	default:
		return false
	}
}

const (
	KidsMinAge     = 4
	KidsMaxAge     = 8
	KidsDefaultAge = 6
)

const (
	KidsAgeTier45 = "4-5"
	KidsAgeTier68 = "6-8"
	KidsAgeTier9  = "9+"
)

func KidsAgeTier(age int) string {
	if age <= 5 {
		return KidsAgeTier45
	}
	if age <= 8 {
		return KidsAgeTier68
	}
	return KidsAgeTier9
}

func (r StartRequest) AgeOrDefault(defaultAge int) int {
	if r.Age == nil {
		return defaultAge
	}
	return *r.Age
}

func IsValidArchetype(a string) bool {
	for _, v := range ValidArchetypes {
		if v == a {
			return true
		}
	}
	return false
}

func IsValidArchetypeForGenre(a, g string) bool {
	if !IsValidArchetype(a) || !IsValidGenre(g) {
		return false
	}
	allowed, ok := GenreArchetypeMatrix[g]
	if !ok {
		// Fallback to true if genre matrix is somehow missing
		return true
	}
	for _, v := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

type GameState struct {
	Health        int            `json:"health"`
	MaxHealth     int            `json:"max_health"`
	Inventory     []string       `json:"inventory"`
	Stats         map[string]int `json:"stats"`
	Bonds         map[string]int `json:"bonds"`
	Karma         int            `json:"karma"`
	FatePoints    int            `json:"fate_points"`
	Reputation    map[string]int `json:"reputation"`
	Flags         []string       `json:"flags"`
	Archetype     string         `json:"archetype"`
	PlotTwists    int            `json:"plot_twists"`
	ChapterNumber int            `json:"chapter_number"`
	// Entities tracks named characters/creatures and their relationships for consistency.
	Entities map[string]Entity `json:"entities"`
}

// IsNewChapter returns true if this turn should start a new chapter.
func (s GameState) IsNewChapter(turnNumber int) bool {
	return turnNumber > 1 && turnNumber%6 == 1
}

// TwistLevel controls the intensity of an injected plot twist.
type TwistLevel string

const (
	TwistNone  TwistLevel = "none"
	TwistMinor TwistLevel = "minor"
	TwistMajor TwistLevel = "major"
	TwistWild  TwistLevel = "wild"
)

type Entity struct {
	Name         string   `json:"name"`
	RelationToPC string   `json:"relation_to_pc"`
	Gender       string   `json:"gender"`
	Role         string   `json:"role"`
	Status       string   `json:"status"`
	Appearance   string   `json:"appearance,omitempty"`
	Traits       []string `json:"traits"`
}

type Session struct {
	ID             string    `json:"id"`
	UserName       string    `json:"user_name"`
	Genre          string    `json:"genre"`
	Age            int       `json:"age"`
	Gender         string    `json:"gender"`
	Archetype      string    `json:"archetype"`
	Seed           string    `json:"seed"`
	State          GameState `json:"state"`
	CurrentChoices []string  `json:"current_choices"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type StartRequest struct {
	Name      string `json:"name"`
	Age       *int   `json:"age"`
	Gender    string `json:"gender"`
	Genre     string `json:"genre"`
	Archetype string `json:"archetype"`
	Seed      string `json:"seed"`
}

type StoryLog struct {
	ID                int       `json:"id"`
	SessionID         string    `json:"session_id"`
	TurnNumber        int       `json:"turn_number"`
	Content           string    `json:"content"`
	ColorCodedContent string    `json:"color_coded_content,omitempty"`
	ChapterTitle      string    `json:"chapter_title"`
	ChoiceMade        string    `json:"choice_made"`
	ImageScene        string    `json:"image_scene,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
	OptionsData       string    `json:"-"`
}
