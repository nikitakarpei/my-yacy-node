package rwipostings_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestScanWordVisitsMatchingPostings(t *testing.T) {
	h := openHarness(t)

	h.admit(t,
		posting("w1", "u1"),
		posting("w1", "u2"),
		posting("w2", "u3"),
	)

	word := yacymodel.WordHash("w1")
	var visited []yacymodel.RWIPosting
	h.scanWord(t, word, func(entry yacymodel.RWIPosting) (bool, error) {
		visited = append(visited, entry)

		return true, nil
	})
	if len(visited) != 2 {
		t.Fatalf("visited %d postings, want 2", len(visited))
	}
	for _, entry := range visited {
		if entry.WordHash != word {
			t.Fatalf("entry word hash = %q, want %q", entry.WordHash, word)
		}
	}
}

func TestPostingReadsBackStoredEntry(t *testing.T) {
	h := openHarness(t)

	h.admit(t,
		posting("w1", "u1"),
	)

	word := yacymodel.WordHash("w1")
	url := urlHash("u1")

	entry, found := h.postingOf(t, word, url)
	if !found {
		t.Fatal("Posting not found")
	}
	if entry.WordHash != word {
		t.Fatalf("entry word hash = %q, want %q", entry.WordHash, word)
	}
	if entry.URLHash != url {
		t.Fatalf("entry url hash = %q, want %q", entry.URLHash, url)
	}
}

func TestPostingMissingIsNotFound(t *testing.T) {
	h := openHarness(t)

	if _, found := h.postingOf(t, yacymodel.WordHash("w1"), urlHash("u1")); found {
		t.Fatal("Posting should not be found")
	}
}

func TestScanWordStopsWhenVisitorStops(t *testing.T) {
	h := openHarness(t)

	h.admit(t,
		posting("w1", "u1"),
		posting("w1", "u2"),
	)

	visited := 0
	h.scanWord(t, yacymodel.WordHash("w1"), func(yacymodel.RWIPosting) (bool, error) {
		visited++

		return false, nil
	})
	if visited != 1 {
		t.Fatalf("visited %d postings, want 1 before stop", visited)
	}
}
