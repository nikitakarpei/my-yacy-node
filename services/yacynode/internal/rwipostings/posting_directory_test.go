package rwipostings_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

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

func TestAdmittingAPostingTwiceReportsAnUpdateNotAnArrival(t *testing.T) {
	h := openHarness(t)

	arrived := posting("w1", "u1")
	h.admit(t, arrived)

	refreshed := arrived
	refreshed.Hits = 7
	h.admit(t, refreshed)

	if len(h.observer.stored) != 1 || h.observer.stored[0] != arrived {
		t.Fatalf("stored notifications = %+v, want only the first admission", h.observer.stored)
	}
	if len(h.observer.updated) != 1 {
		t.Fatalf("updated notifications = %d, want 1", len(h.observer.updated))
	}
	update := h.observer.updated[0]
	if update.previous != arrived || update.current != refreshed {
		t.Fatalf("update carried %+v, want the posting as it was and as it is", update)
	}
}
