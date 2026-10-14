package rwipostings_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPurgePostingDropsPosting(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t)

	h.admit(t,
		posting("w1", "u1"),
		posting("w1", "u2"),
		posting("w2", "u1"),
	)

	word := yacymodel.WordHash("w1")
	url := urlHash("u1")
	var wasPurged bool
	if err := h.vault.Update(ctx, func(tx *vault.Txn) error {
		var err error
		wasPurged, err = h.purger.PurgePosting(tx, word, url)
		if err != nil {
			return fmt.Errorf("purge posting: %w", err)
		}

		return nil
	}); err != nil {
		t.Fatalf("PurgePosting: %v", err)
	}
	if !wasPurged {
		t.Fatal("PurgePosting reported nothing deleted, want the posting dropped")
	}

	if _, found := h.postingOf(t, word, url); found {
		t.Fatal("PostingOf found the purged posting")
	}
	if _, found := h.postingOf(t, word, urlHash("u2")); !found {
		t.Fatal("PostingOf lost a posting of the word that was not purged")
	}
}

func TestPurgingAPostingThatIsNotHeldReportsNothingPurged(t *testing.T) {
	h := openHarness(t)

	h.admit(t, posting("w1", "u1"))

	var wasPurged bool
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		var err error
		wasPurged, err = h.purger.PurgePosting(
			tx, yacymodel.WordHash("w2"), urlHash("u2"),
		)
		if err != nil {
			return fmt.Errorf("purge posting: %w", err)
		}

		return nil
	}); err != nil {
		t.Fatalf("PurgePosting: %v", err)
	}

	if wasPurged {
		t.Fatal("PurgePosting reported a deletion, want none")
	}
}

func TestPurgingAWordInAPartitionDropsOnlyThosePostings(t *testing.T) {
	h := openHarness(t)
	words := neighbouringWordHashes(t)
	for _, word := range words {
		for document := range documentsPerWord {
			h.admit(t, postingOfWordOn(word, urlHash(strconv.Itoa(document))))
		}
	}
	word := words[1]
	partition, purgedPostings := busiestPartitionOf(word)

	var amountOfPurgedPostings int
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		var err error
		amountOfPurgedPostings, err = h.purger.PurgePostingsOfWordIn(tx, word, partition)

		return err
	}); err != nil {
		t.Fatalf("PurgePostingsOfWordIn: %v", err)
	}

	if amountOfPurgedPostings != len(purgedPostings) {
		t.Fatalf("purged postings = %d, want %d", amountOfPurgedPostings, len(purgedPostings))
	}
	if leftPostings := h.postingsOfWordIn(t, word, partition, 1); len(leftPostings) != 0 {
		t.Fatalf("postings left in the purged partition = %v, want none", leftPostings)
	}
	for _, neighbour := range []yacymodel.Hash{words[0], words[2]} {
		keptPostings := h.postingsOfWordIn(t, neighbour, partition, documentsPerWord)
		if len(keptPostings) != len(purgedPostings) {
			t.Fatalf(
				"postings of a neighbour word = %d, want %d",
				len(keptPostings),
				len(purgedPostings),
			)
		}
	}
	for document := range documentsPerWord {
		url := urlHash(strconv.Itoa(document))
		if ringPartitions.PartitionOf(url) == partition {
			continue
		}
		if _, found := h.postingOf(t, word, url); !found {
			t.Fatalf("posting of the word in another partition on %v was purged", url)
		}
	}
}
