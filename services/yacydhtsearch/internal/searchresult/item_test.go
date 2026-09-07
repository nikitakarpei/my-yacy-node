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
	item, ok := searchresult.ItemFrom(yacymodel.URLMetadata{
		Address:        "https://example.org/weather",
		Title:          "Weather",
		Snippet:        "",
		Modified:       yacymodel.Some(modified),
		FaviconAddress: "https://example.org/icon.png",
	})
	if !ok {
		t.Fatal("ItemFrom refused metadata that names a reachable address")
	}
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
	item, _ := searchresult.ItemFrom(yacymodel.URLMetadata{
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

	item, _ := searchresult.ItemFrom(yacymodel.URLMetadata{Address: "https://example.org/"})

	if _, ok := item.PublishedAt.Get(); ok {
		t.Fatalf("PublishedAt = %+v, want none", item.PublishedAt)
	}
}

func TestItemFromRefusesAnAddressThatIsNotOne(t *testing.T) {
	t.Parallel()

	if _, ok := searchresult.ItemFrom(yacymodel.URLMetadata{Address: "://"}); ok {
		t.Fatal("ItemFrom accepted an address that is not a URL")
	}
}
