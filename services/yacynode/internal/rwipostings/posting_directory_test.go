package rwipostings_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPostingMissingIsNotFound(t *testing.T) {
	h := openHarness(t)

	if _, found := h.postingOf(t, yacymodel.WordHash("w1"), urlHash("u1")); found {
		t.Fatal("Posting should not be found")
	}
}

func TestWritingAChangedPostingReplacesTheStoredOne(t *testing.T) {
	h := openHarness(t)

	arrived := posting("w1", "u1")
	h.writeEach(t, arrived)
	if stored, found := h.postingOf(t, arrived.WordHash, arrived.URLHash); !found ||
		stored != arrived {
		t.Fatalf("PostingOf = %+v, %v, want the posting as it arrived", stored, found)
	}

	refreshed := arrived
	refreshed.Hits = 7
	h.writeEach(t, refreshed)

	if stored, _ := h.postingOf(t, arrived.WordHash, arrived.URLHash); stored != refreshed {
		t.Fatalf("PostingOf = %+v, want the posting as it is", stored)
	}
}

func TestPostingsOfWordInAPartitionHoldOnlyThatWordAndPartition(t *testing.T) {
	h := openHarness(t)
	words := neighbouringWordHashes(t)
	for _, word := range words {
		for document := range documentsPerWord {
			h.writeEach(t, postingOfWordOn(word, urlHash(strconv.Itoa(document))))
		}
	}
	word := words[1]
	partition, wantedPostings := busiestPartitionOf(t, word)

	visitedPostings := h.postingsOfWordIn(t, word, partition, len(wantedPostings)+1)

	if len(visitedPostings) != len(wantedPostings) ||
		slices.ContainsFunc(wantedPostings, func(wanted yacymodel.RWIPosting) bool {
			return !slices.Contains(visitedPostings, wanted)
		}) {
		t.Fatalf("PostingsOfWordIn = %v, want %v", visitedPostings, wantedPostings)
	}
}

func TestPostingsOfWordInAPartitionStopWhenTheVisitorAsks(t *testing.T) {
	h := openHarness(t)
	word := neighbouringWordHashes(t)[1]
	for document := range documentsPerWord {
		h.writeEach(t, postingOfWordOn(word, urlHash(strconv.Itoa(document))))
	}
	partition, wantedPostings := busiestPartitionOf(t, word)

	visitedPostings := h.postingsOfWordIn(t, word, partition, 1)

	if len(visitedPostings) != 1 || !slices.Contains(wantedPostings, visitedPostings[0]) {
		t.Fatalf("PostingsOfWordIn = %v, want one of %v", visitedPostings, wantedPostings)
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

func busiestPartitionOf(
	t *testing.T,
	word yacymodel.Hash,
) (uint, []yacymodel.RWIPosting) {
	t.Helper()

	postingsPerPartition := make(map[uint][]yacymodel.RWIPosting)
	for document := range documentsPerWord {
		url := urlHash(strconv.Itoa(document))
		partition := ringPartitions(t).PartitionOf(url)
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
	return busiestPartition, postingsPerPartition[busiestPartition]
}
