package queryanswers_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const countedWord = "berlin"

func foundDocumentOfMetadata(
	t *testing.T, metadata ...yacymodel.URLMetadata,
) queryanswers.FoundDocument {
	t.Helper()

	metadataReplicas := make([]queryanswers.MetadataReplica, 0, len(metadata))
	for _, reported := range metadata {
		metadataReplicas = append(metadataReplicas, queryanswers.MetadataReplica{
			Holder: yacymodel.WordHash("a holder"), Metadata: reported,
		})
	}

	return queryanswers.FoundDocumentOf(
		documentOf(t, "https://example.org/"),
		metadataReplicas,
		nil,
	)
}

func replicaOfCountedWord(
	hits int, localLinks int, externalLinks int,
) queryanswers.PostingReplica {
	return queryanswers.PostingReplica{
		Holder: yacymodel.WordHash("a holder"),
		Word:   yacymodel.WordHash(countedWord),
		Posting: yacymodel.RWIPosting{
			Hits:          hits,
			TextWords:     1200,
			LocalLinks:    localLinks,
			ExternalLinks: externalLinks,
		},
	}
}

func factsOfDocumentOf(
	t *testing.T, replicas ...queryanswers.PostingReplica,
) queryanswers.DocumentFacts {
	t.Helper()

	return queryanswers.FoundDocumentOf(
		documentOf(t, "https://berlin.example/"), nil, replicas,
	).Facts
}

func TestFoundDocumentCarriesWhatAPeerReported(t *testing.T) {
	t.Parallel()

	modified := yacymodel.NewCalendarDay(2026, time.March, 4)
	foundDocument := foundDocumentOfMetadata(t, yacymodel.URLMetadata{
		Address:        "https://example.org/weather",
		Title:          "Weather",
		Snippet:        "Rain in Berlin.",
		Modified:       yacymodel.Some(modified),
		FaviconAddress: "https://example.org/icon.png",
	})

	if foundDocument.Address != "https://example.org/weather" ||
		foundDocument.Title != "Weather" || foundDocument.Snippet != "Rain in Berlin." ||
		foundDocument.FaviconAddress != "https://example.org/icon.png" {
		t.Fatalf("the found document reads %+v, want what the peer reported", foundDocument)
	}
	published, _ := foundDocument.PublishedAt.Get()
	if !published.Equal(modified.Time()) {
		t.Fatalf("PublishedAt = %v, want %v", published, modified.Time())
	}
}

func TestFoundDocumentIsShownAsTheFirstPeerReportedIt(t *testing.T) {
	t.Parallel()

	foundDocument := foundDocumentOfMetadata(
		t,
		yacymodel.URLMetadata{
			Address: "https://example.org/weather", Title: "Weather", Snippet: "Rain in Berlin.",
		},
		yacymodel.URLMetadata{
			Address: "https://example.org/later", Title: "Later", Snippet: "Sun in Berlin.",
		},
	)

	if foundDocument.Address != "https://example.org/weather" ||
		foundDocument.Title != "Weather" || foundDocument.Snippet != "Rain in Berlin." {
		t.Fatalf("the found document reads %+v, want what the first peer reported", foundDocument)
	}
	if len(foundDocument.MetadataReplicas) != 2 {
		t.Fatalf(
			"the found document holds %+v, want the metadata of both peers",
			foundDocument.MetadataReplicas,
		)
	}
}

func TestFoundDocumentFallsBackToDayThePeerLoadedIt(t *testing.T) {
	t.Parallel()

	loaded := yacymodel.NewCalendarDay(2025, time.December, 31)
	foundDocument := foundDocumentOfMetadata(
		t, yacymodel.URLMetadata{Address: "https://example.org/", Loaded: yacymodel.Some(loaded)},
	)

	published, ok := foundDocument.PublishedAt.Get()
	if !ok || !published.Equal(loaded.Time()) {
		t.Fatalf("PublishedAt = %v %v, want %v", published, ok, loaded.Time())
	}
}

func TestFoundDocumentLeavesPublicationDayUnsetWhenThePeerNamedNone(t *testing.T) {
	t.Parallel()

	foundDocument := foundDocumentOfMetadata(
		t, yacymodel.URLMetadata{Address: "https://example.org/"},
	)

	if _, ok := foundDocument.PublishedAt.Get(); ok {
		t.Fatalf("PublishedAt = %+v, want none", foundDocument.PublishedAt)
	}
}

func TestFoundDocumentKeepsNameItWasFoundUnder(t *testing.T) {
	t.Parallel()

	named := documentOf(t, "https://example.org/")
	foundDocument := foundDocumentOfMetadata(
		t, yacymodel.URLMetadata{Address: "https://example.org/somewhere-else"},
	)

	if foundDocument.Hash != named {
		t.Fatalf("Hash = %q, want the name it was found under %q", foundDocument.Hash, named)
	}
}

func TestPostingOfWordCountsItsHitsAndLinksOfDocument(t *testing.T) {
	t.Parallel()

	facts := factsOfDocumentOf(t, replicaOfCountedWord(7, 20, 9))

	if facts.HitsPerQueryWord[yacymodel.WordHash(countedWord)] != 7 ||
		facts.AmountOfLinks.OrElse(0) != 29 {
		t.Fatalf("the document reads %+v, want the 7 hits and the 29 links the peer counted", facts)
	}
}

func TestOnlyFirstPostingOfWordCountsForDocument(t *testing.T) {
	t.Parallel()

	facts := factsOfDocumentOf(
		t, replicaOfCountedWord(7, 20, 9), replicaOfCountedWord(3, 1, 0),
	)

	if facts.HitsPerQueryWord[yacymodel.WordHash(countedWord)] != 7 ||
		facts.AmountOfLinks.OrElse(0) != 29 {
		t.Fatalf("the document reads %+v, want what the first holder counted", facts)
	}
}

func TestNoPeerCountsWordsOrQueryPhrasesOfDocument(t *testing.T) {
	t.Parallel()

	facts := factsOfDocumentOf(t, replicaOfCountedWord(7, 20, 9))

	if facts.AmountOfWords.Present() || facts.QueryPhraseHits.Present() {
		t.Fatalf(
			"the document reads %+v, want no amount of words and no query phrase hits of a peer",
			facts,
		)
	}
}

func TestDocumentNoPeerSentPostingOfCountsNoFact(t *testing.T) {
	t.Parallel()

	foundDocument := foundDocumentOfMetadata(
		t, yacymodel.URLMetadata{Address: "https://example.org/"},
	)

	if len(foundDocument.Facts.HitsPerQueryWord) != 0 ||
		foundDocument.Facts.AmountOfLinks.Present() {
		t.Fatalf("the document reads %+v, want no fact at all", foundDocument.Facts)
	}
}
