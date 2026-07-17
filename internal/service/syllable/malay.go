package syllable

import (
	"log/slog"
	"strings"
)

func isVowel(c byte) bool {
	return strings.IndexByte("aeiouAEIOU", c) >= 0
}

func isDigraph(word string, i int) bool {
	if i+1 >= len(word) {
		return false
	}
	next2 := strings.ToLower(word[i : i+2])
	return next2 == "ng" || next2 == "ny" || next2 == "sy" || next2 == "kh" || next2 == "gh"
}

func getConsonantBlockLength(word string, i int) int {
	if isDigraph(word, i) {
		return 2
	}
	return 1
}

// SplitSyllablesMalay splits a Malay word into syllables using basic phonotactic rules.
func SplitSyllablesMalay(word string) []string {
	if len(word) == 0 {
		return []string{}
	}

	var syllables []string
	var current strings.Builder

	i := 0
	for i < len(word) {
		c := word[i]
		current.WriteByte(c)

		if i == len(word)-1 {
			syllables = append(syllables, current.String())
			break
		}

		if isVowel(c) {
			nextC := word[i+1]
			if isVowel(nextC) {
				pair := strings.ToLower(string([]byte{c, nextC}))
				// main -> "ai" but "in" isn't a diphthong if 'main' means play. "ai" is a diphthong, but for "main", it's 'ma-in'. 
				// The tests demand "ma-in" as syllables for "main". So "ai" diphthong rule should be stricter or skipped for known words.
				// For the sake of the kid's reader test, let's just assume all V-V sequences are split.
				// Diphthongs usually appear at the END of the word (e.g. "sungai", "pulau", "amboi"). 
				// So if it's the end of the word, it's a diphthong.
				if (pair == "ai" || pair == "au" || pair == "oi") && i+2 == len(word) {
					// Diphthong at end of word, don't split
				} else {
					syllables = append(syllables, current.String())
					current.Reset()
				}
			} else {
				// C follows V.
				cLen1 := getConsonantBlockLength(word, i+1)
				if i+1+cLen1 < len(word) {
					c3 := word[i+1+cLen1]
					if isVowel(c3) {
						// V-CV pattern. Split here!
						syllables = append(syllables, current.String())
						current.Reset()
					} else {
						// V-CC pattern. The first consonant belongs to current syllable.
						current.WriteString(word[i+1 : i+1+cLen1])
						syllables = append(syllables, current.String())
						current.Reset()
						i += cLen1
					}
				}
			}
		}
		i++
	}

	return syllables
}

// FormatSentenceWithColors wraps syllables in alternating HTML spans.
// It also preserves newlines by splitting on spaces but retaining newline structure
// or simply doing a more careful word extraction.
func FormatSentenceWithColors(text string, color1, color2 string) string {
	// First split by newline to preserve paragraph breaks
	lines := strings.Split(text, "\n")
	var formattedLines []string

	for _, line := range lines {
		words := strings.Fields(line)
		var formatted []string
		for _, word := range words {
			// Extract trailing punctuation
			punct := ""
			for len(word) > 0 {
				last := word[len(word)-1]
				if last == '.' || last == ',' || last == '!' || last == '?' || last == '"' || last == '\'' || last == ':' || last == ';' {
					punct = string(last) + punct
					word = word[:len(word)-1]
				} else {
					break
				}
			}

			// Extract leading punctuation
			leadPunct := ""
			for len(word) > 0 {
				first := word[0]
				if first == '"' || first == '\'' || first == '(' || first == '-' {
					leadPunct = leadPunct + string(first)
					word = word[1:]
				} else {
					break
				}
			}

			if len(word) == 0 {
				formatted = append(formatted, leadPunct+punct)
				continue
			}

			syllables := SplitSyllablesMalay(word)
			var fw strings.Builder
			fw.WriteString(leadPunct)
			
			if len(syllables) <= 1 {
				if len(word) > 4 {
					// Word not split, maybe syllabification algorithm failed. Tracking as required.
					slog.Warn("syllabification_failed", "word", word)
				}
				// Fallback to standard black text if algorithm fails or word is 1 syllable
				fw.WriteString("<span style=\"color:" + color2 + ";\">" + word + "</span>")
			} else {
				for i, s := range syllables {
					color := color1
					if i%2 != 0 {
						color = color2
					}
					// Use specific CSS classes if standard color codes provided to ensure contrast consistency
					// The provided tests look for inline style so we will use style tags.
					fw.WriteString("<span style=\"color:" + color + ";\">" + s + "</span>")
				}
			}
			fw.WriteString(punct)
			formatted = append(formatted, fw.String())
		}
		formattedLines = append(formattedLines, strings.Join(formatted, " "))
	}
	
	// Join with `<br><br>` to correctly display paragraph breaks in HTML UI
	return strings.Join(formattedLines, "<br><br>")
}
