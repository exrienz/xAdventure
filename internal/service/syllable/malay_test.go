package syllable

import (
	"reflect"
	"testing"
)

func TestSplitSyllablesMalay(t *testing.T) {
	tests := []struct {
		word     string
		expected []string
	}{
		{"buku", []string{"bu", "ku"}},
		{"kereta", []string{"ke", "re", "ta"}},
		{"makan", []string{"ma", "kan"}},
		{"pintu", []string{"pin", "tu"}},
		{"main", []string{"ma", "in"}},
		{"sungai", []string{"su", "ngai"}},
		{"nyanyi", []string{"nya", "nyi"}},
	}

	for _, tt := range tests {
		got := SplitSyllablesMalay(tt.word)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("SplitSyllablesMalay(%q) = %v; want %v", tt.word, got, tt.expected)
		}
	}
}

func TestFormatSentenceWithColors(t *testing.T) {
	sentence := "Buku ini merah."
	expected := `<span style="color:black;">Bu</span><span style="color:red;">ku</span> <span style="color:black;">i</span><span style="color:red;">ni</span> <span style="color:black;">me</span><span style="color:red;">rah</span>.`
	
	got := FormatSentenceWithColors(sentence, "black", "red")
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}
