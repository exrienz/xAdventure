package domain

import "testing"

func TestKidsAgeTier(t *testing.T) {
	tests := map[int]string{
		4: KidsAgeTier45,
		5: KidsAgeTier45,
		6: KidsAgeTier68,
		8: KidsAgeTier68,
	}

	for age, want := range tests {
		if got := KidsAgeTier(age); got != want {
			t.Fatalf("KidsAgeTier(%d) = %q; want %q", age, got, want)
		}
	}
}

func TestKidsAgeOrDefault(t *testing.T) {
	req := StartRequest{Genre: "Pengembaraan"}
	if got := req.AgeOrDefault(KidsDefaultAge); got != KidsDefaultAge {
		t.Fatalf("missing age = %d; want %d", got, KidsDefaultAge)
	}

	age := 4
	req.Age = &age
	if got := req.AgeOrDefault(KidsDefaultAge); got != 4 {
		t.Fatalf("present age = %d; want 4", got)
	}
}

func TestIsKidsGenre(t *testing.T) {
	for _, genre := range []string{"Pengembaraan", "Fantasi", "Cerita Haiwan", "Kisah Dongeng", "Kelakar", "Persahabatan", "Sains & Teknologi", "Mistik/Misteri"} {
		if !IsKidsGenre(genre) {
			t.Fatalf("expected %q to be a kids genre", genre)
		}
	}
	for _, genre := range []string{"Kids", "Fabel", "Humor", "Misteri Kanak-kanak"} {
		if IsKidsGenre(genre) {
			t.Fatalf("expected legacy genre %q to no longer be a kids genre", genre)
		}
	}
	if IsKidsGenre("Mystery") {
		t.Fatal("Mystery should not be treated as a kids genre")
	}
	// Verify all genres in ValidGenres that are kids pass IsKidsGenre.
	for _, genre := range ValidGenres {
		if IsKidsGenre(genre) {
			// Must also be in GenreArchetypeMatrix.
			if _, ok := GenreArchetypeMatrix[genre]; !ok {
				t.Fatalf("kids genre %q missing from GenreArchetypeMatrix", genre)
			}
		}
	}
}
