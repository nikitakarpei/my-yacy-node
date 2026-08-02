package rwipostings

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestScanWordVisitsMatchingPostings(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t)

	h.admit(t,
		posting("w1", "u1"),
		posting("w1", "u2"),
		posting("w2", "u3"),
	)

	word := yacymodel.WordHash("w1")
	var visited []yacymodel.RWIPosting
	err := h.index.ScanWord(ctx, word, func(entry yacymodel.RWIPosting) (bool, error) {
		visited = append(visited, entry)

		return true, nil
	})
	if err != nil {
		t.Fatalf("ScanWord: %v", err)
	}
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
	ctx := context.Background()
	h := openHarness(t)

	h.admit(t,
		posting("w1", "u1"),
	)

	word := yacymodel.WordHash("w1")
	url := urlHash("u1")

	entry, found, err := h.index.PostingOf(ctx, word, url)
	if err != nil {
		t.Fatalf("Posting: %v", err)
	}
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
	ctx := context.Background()
	h := openHarness(t)

	_, found, err := h.index.PostingOf(ctx, yacymodel.WordHash("w1"), urlHash("u1"))
	if err != nil {
		t.Fatalf("Posting: %v", err)
	}
	if found {
		t.Fatal("Posting should not be found")
	}
}

func TestScanWordStopsWhenVisitorStops(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t)

	h.admit(t,
		posting("w1", "u1"),
		posting("w1", "u2"),
	)

	visited := 0
	err := h.index.ScanWord(
		ctx,
		yacymodel.WordHash("w1"),
		func(yacymodel.RWIPosting) (bool, error) {
			visited++

			return false, nil
		},
	)
	if err != nil {
		t.Fatalf("ScanWord: %v", err)
	}
	if visited != 1 {
		t.Fatalf("visited %d postings, want 1 before stop", visited)
	}
}
