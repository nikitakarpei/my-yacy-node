package searchresult_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestItemFromCarriesWhatAPeerReported(t *testing.T) {
	t.Parallel()

	modified := yacymodel.NewCalendarDay(2026, time.March, 4)
	item := searchresult.ItemFrom(yacymodel.URLMetadata{
		Address:        "https://example.org/weather",
		Title:          "Weather",
		Snippet:        "",
		Modified:       yacymodel.Some(modified),
		FaviconAddress: "https://example.org/icon.png",
	})
	if item.Address != "https://example.org/weather" || item.Title != "Weather" {
		t.Fatalf("ItemFrom = %+v, want the reported address and title", item)
	}
	if item.ImageAddress != "https://example.org/icon.png" {
		t.Fatalf("ImageAddress = %q, want the reported favicon", item.ImageAddress)
	}
	published, _ := item.PublishedAt.Get()
	if !published.Equal(modified.Time()) {
		t.Fatalf("PublishedAt = %v, want %v", published, modified.Time())
	}
}

func TestItemFromFallsBackToTheDayThePeerLoadedIt(t *testing.T) {
	t.Parallel()

	loaded := yacymodel.NewCalendarDay(2025, time.December, 31)
	item := searchresult.ItemFrom(yacymodel.URLMetadata{
		Address: "https://example.org/",
		Loaded:  yacymodel.Some(loaded),
	})

	published, ok := item.PublishedAt.Get()
	if !ok || !published.Equal(loaded.Time()) {
		t.Fatalf("PublishedAt = %v %v, want %v", published, ok, loaded.Time())
	}
}

func TestItemFromLeavesThePublicationDayUnsetWhenThePeerNamedNone(t *testing.T) {
	t.Parallel()

	item := searchresult.ItemFrom(yacymodel.URLMetadata{Address: "https://example.org/"})

	if _, ok := item.PublishedAt.Get(); ok {
		t.Fatalf("PublishedAt = %+v, want none", item.PublishedAt)
	}
}

func TestItemFromKeepsTheNameThePeerHoldsTheDocumentUnder(t *testing.T) {
	t.Parallel()

	named, err := yacymodel.ParseURLHash("TULSZfg4-80c")
	if err != nil {
		t.Fatalf("ParseURLHash: %v", err)
	}

	item := searchresult.ItemFrom(yacymodel.URLMetadata{
		Hash:    named,
		Address: "https://example.org/somewhere-else",
	})

	if item.Hash != named {
		t.Fatalf("Hash = %q, want the name the peer reported %q", item.Hash, named)
	}
}
