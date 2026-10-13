package rwipostings_test

import (
	"bytes"
	"slices"
	"strconv"
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

func TestPostingsOfWordInAPartitionHoldOnlyThatWordAndPartitionInKeyOrder(t *testing.T) {
	h := openHarness(t)
	words := neighbouringWordHashes(t)
	for _, word := range words {
		for document := range documentsPerWord {
			h.admit(t, postingOfWordOn(word, urlHash(strconv.Itoa(document))))
		}
	}
	word := words[1]
	partition, wantedPostings := busiestPartitionOf(word)

	visitedPostings := h.postingsOfWordIn(t, word, partition, len(wantedPostings)+1)

	if !slices.Equal(visitedPostings, wantedPostings) {
		t.Fatalf("PostingsOfWordIn = %v, want %v", visitedPostings, wantedPostings)
	}
}

func TestPostingsOfWordInAPartitionStopWhenTheVisitorAsks(t *testing.T) {
	h := openHarness(t)
	word := neighbouringWordHashes(t)[1]
	for document := range documentsPerWord {
		h.admit(t, postingOfWordOn(word, urlHash(strconv.Itoa(document))))
	}
	partition, wantedPostings := busiestPartitionOf(word)

	visitedPostings := h.postingsOfWordIn(t, word, partition, 1)

	if !slices.Equal(visitedPostings, wantedPostings[:1]) {
		t.Fatalf("PostingsOfWordIn = %v, want %v", visitedPostings, wantedPostings[:1])
	}
}

const documentsPerWord = 64

func neighbouringWordHashes(t *testing.T) []yacymodel.Hash {
	t.Helper()

	words := make([]yacymodel.Hash, 0, 3)
	for _, text := range []string{"neighbourAAz", "neighbourAA0", "neighbourAA1"} {
		word, err := yacymodel.ParseHash(text)
		if err != nil {
			t.Fatalf("ParseHash(%s): %v", text, err)
		}
		words = append(words, word)
	}

	return words
}

func postingOfWordOn(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:   word,
		URLHash:    url,
		Language:   yacymodel.LanguageOfUndeclaredDocument,
		LocalLinks: 1,
		Hits:       1,
	}
}

func busiestPartitionOf(word yacymodel.Hash) (uint, []yacymodel.RWIPosting) {
	postingsPerPartition := make(map[uint][]yacymodel.RWIPosting)
	for document := range documentsPerWord {
		url := urlHash(strconv.Itoa(document))
		partition := ringPartitions.PartitionOf(url)
		postingsPerPartition[partition] = append(
			postingsPerPartition[partition],
			postingOfWordOn(word, url),
		)
	}

	var busiestPartition uint
	for partition, postings := range postingsPerPartition {
		if len(postings) > len(postingsPerPartition[busiestPartition]) {
			busiestPartition = partition
		}
	}
	busiestPostings := postingsPerPartition[busiestPartition]
	slices.SortFunc(busiestPostings, func(a, b yacymodel.RWIPosting) int {
		return bytes.Compare(a.URLHash.Bytes(), b.URLHash.Bytes())
	})

	return busiestPartition, busiestPostings
}
