package rwipostings_test

import (
	"slices"
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

func TestAdmittingAChangedPostingPurgesTheOneItReplaces(t *testing.T) {
	h := openHarness(t)

	arrived := posting("w1", "u1")
	h.admit(t, arrived)

	refreshed := arrived
	refreshed.Hits = 7
	h.admit(t, refreshed)

	wantedNotifications := []string{storedNotification, purgedNotification, storedNotification}
	if !slices.Equal(h.observer.notificationsInOrder, wantedNotifications) {
		t.Fatalf(
			"notifications = %v, want the replaced posting purged before the one that replaces it",
			h.observer.notificationsInOrder,
		)
	}
	if len(h.observer.purged) != 1 || h.observer.purged[0] != arrived {
		t.Fatalf("purged notifications = %+v, want the posting as it was", h.observer.purged)
	}
	if len(h.observer.stored) != 2 || h.observer.stored[1] != refreshed {
		t.Fatalf("stored notifications = %+v, want the posting as it is", h.observer.stored)
	}
}

func TestAdmittingAnEqualPostingAgainNotifiesNobody(t *testing.T) {
	h := openHarness(t)

	arrived := posting("w1", "u1")
	h.admit(t, arrived)
	h.admit(t, arrived)

	if !slices.Equal(h.observer.notificationsInOrder, []string{storedNotification}) {
		t.Fatalf(
			"notifications = %v, want only the first admission",
			h.observer.notificationsInOrder,
		)
	}
}
