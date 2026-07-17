package service

import (
	"strings"
	"unicode"
)

// SafetyFilter checks generated story text for disallowed content patterns.
// It is a lightweight heuristic; it does not replace provider-side moderation.
type SafetyFilter struct {
	blockedWords []string
}

func NewSafetyFilter() *SafetyFilter {
	return &SafetyFilter{
		blockedWords: []string{
			"sexual", "sex", "rape", "molest", "nude", "naked", "porn", "erotic",
			"nsfw", "gore porn", "torture porn", "explicit sexual", "sexual assault",
			"fuck", "shit", "bitch", "damn", "asshole", "cunt", "dick", "pussy", "cock",
			"bastard", "slut", "whore", "retard", "nigger", "faggot", "chink",
		},
	}
}

// IsSafeName checks whether a generated name contains profanity or slurs.
func (sf *SafetyFilter) IsSafeName(name string) bool {
	lower := strings.ToLower(name)
	for _, word := range sf.blockedWords {
		if strings.Contains(lower, word) {
			return false
		}
	}
	return true
}

// IsSafe checks story text for disallowed content patterns.
func (sf *SafetyFilter) IsSafe(text string) bool {
	lower := strings.ToLower(text)
	for _, word := range sf.blockedWords {
		if strings.Contains(lower, word) {
			return false
		}
	}
	return true
}

func (sf *SafetyFilter) Sanitize(text string) string {
	if sf.IsSafe(text) {
		return text
	}
	// Replace blocked words with asterisks.
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for _, word := range words {
		lower := strings.ToLower(word)
		for _, blocked := range sf.blockedWords {
			if lower == blocked {
				text = strings.ReplaceAll(text, word, strings.Repeat("*", len(word)))
			}
		}
	}
	return text
}
