package documentshortlist_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentshortlist"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
)

func documentOf(
	document string,
	relevance searchrelevance.Relevance,
	termSpread int,
) documentshortlist.RankedDocument {
	return documentshortlist.RankedDocument{
		JoinedPosting: yacymodel.RWIPosting{URLHash: searchtest.URLHashFor(document)},
		Relevance:     relevance,
		TermSpread:    termSpread,
	}
}

func wantsDocuments(
	t *testing.T,
	shortlist *documentshortlist.Shortlist,
	wanted ...string,
) {
	t.Helper()

	wantedDocuments := make([]yacymodel.URLHash, 0, len(wanted))
	for _, document := range wanted {
		wantedDocuments = append(wantedDocuments, searchtest.URLHashFor(document))
	}
	if !slices.Equal(documentsOf(shortlist), wantedDocuments) {
		t.Errorf("documents = %v, want %v", documentsOf(shortlist), wantedDocuments)
	}
}

func documentsOf(shortlist *documentshortlist.Shortlist) []yacymodel.URLHash {
	documents := shortlist.InRelevanceOrder()
	held := make([]yacymodel.URLHash, 0, len(documents))
	for _, document := range documents {
		held = append(held, document.JoinedPosting.URLHash)
	}

	return held
}

func TestTheMostRelevantDocumentComesFirst(t *testing.T) {
	shortlist := documentshortlist.New(10)

	shortlist.Place(documentOf("u1", 3, 0))
	shortlist.Place(documentOf("u2", 9, 0))
	shortlist.Place(documentOf("u3", 6, 0))

	wantsDocuments(t, shortlist, "u2", "u3", "u1")
}

func TestTheShortlistDropsWhatDoesNotFitTheResultsTheSearchAsksFor(t *testing.T) {
	shortlist := documentshortlist.New(2)

	shortlist.Place(documentOf("u1", 3, 0))
	shortlist.Place(documentOf("u2", 9, 0))
	shortlist.Place(documentOf("u3", 6, 0))
	shortlist.Place(documentOf("u4", 1, 0))

	wantsDocuments(t, shortlist, "u2", "u3")
}

func TestTheShortlistHoldsEveryDocumentWhenTheSearchAsksForNoAmount(t *testing.T) {
	shortlist := documentshortlist.New(0)

	shortlist.Place(documentOf("u1", 3, 0))
	shortlist.Place(documentOf("u2", 9, 0))
	shortlist.Place(documentOf("u3", 6, 0))

	wantsDocuments(t, shortlist, "u2", "u3", "u1")
	if shortlist.IsFull() {
		t.Error("the shortlist is full, want room for every document")
	}
}

func TestTheClosestTermsComeFirstAmongDocumentsOfOneRelevance(t *testing.T) {
	shortlist := documentshortlist.New(10)

	shortlist.Place(documentOf("u1", 5, 8))
	shortlist.Place(documentOf("u2", 5, 2))

	wantsDocuments(t, shortlist, "u2", "u1")
}

func TestTheURLHashOrdersDocumentsOfOneRelevanceAndTermSpread(t *testing.T) {
	shortlist := documentshortlist.New(10)

	shortlist.Place(documentOf("u2", 5, 4))
	shortlist.Place(documentOf("u1", 5, 4))

	wantsDocuments(t, shortlist, "u1", "u2")
}

func TestAnEmptyShortlistIsNotFullAndHoldsNoRelevance(t *testing.T) {
	shortlist := documentshortlist.New(2)

	if shortlist.IsFull() {
		t.Error("the shortlist is full, want it empty")
	}
	if shortlist.LowestRelevance() != 0 {
		t.Errorf("lowest relevance = %v, want none", shortlist.LowestRelevance())
	}
	if len(documentsOf(shortlist)) != 0 {
		t.Errorf("documents = %v, want none", documentsOf(shortlist))
	}
}

func TestTheLowestRelevanceIsTheRelevanceOfTheLastDocumentTheShortlistHolds(t *testing.T) {
	shortlist := documentshortlist.New(2)

	shortlist.Place(documentOf("u1", 3, 0))
	if shortlist.IsFull() {
		t.Error("the shortlist is full, want room for one more document")
	}
	shortlist.Place(documentOf("u2", 9, 0))

	if !shortlist.IsFull() {
		t.Error("the shortlist has room, want it full")
	}
	if shortlist.LowestRelevance() != 3 {
		t.Errorf("lowest relevance = %v, want 3 of the last document", shortlist.LowestRelevance())
	}
}
