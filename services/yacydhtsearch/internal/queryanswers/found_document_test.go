package queryanswers_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestAFoundDocumentCarriesWhatAPeerReported(t *testing.T) {
	t.Parallel()

	modified := yacymodel.NewCalendarDay(2026, time.March, 4)
	foundDocument := queryanswers.FoundDocumentFrom(yacymodel.URLMetadata{
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

func TestAFoundDocumentFallsBackToTheDayThePeerLoadedIt(t *testing.T) {
	t.Parallel()

	loaded := yacymodel.NewCalendarDay(2025, time.December, 31)
	foundDocument := queryanswers.FoundDocumentFrom(yacymodel.URLMetadata{
		Address: "https://example.org/",
		Loaded:  yacymodel.Some(loaded),
	})

	published, ok := foundDocument.PublishedAt.Get()
	if !ok || !published.Equal(loaded.Time()) {
		t.Fatalf("PublishedAt = %v %v, want %v", published, ok, loaded.Time())
	}
}

func TestAFoundDocumentLeavesThePublicationDayUnsetWhenThePeerNamedNone(t *testing.T) {
	t.Parallel()

	foundDocument := queryanswers.FoundDocumentFrom(
		yacymodel.URLMetadata{Address: "https://example.org/"},
	)

	if _, ok := foundDocument.PublishedAt.Get(); ok {
		t.Fatalf("PublishedAt = %+v, want none", foundDocument.PublishedAt)
	}
}

func TestAFoundDocumentKeepsTheNameThePeerHoldsTheDocumentUnder(t *testing.T) {
	t.Parallel()

	named, err := yacymodel.ParseURLHash("TULSZfg4-80c")
	if err != nil {
		t.Fatalf("ParseURLHash: %v", err)
	}
	foundDocument := queryanswers.FoundDocumentFrom(yacymodel.URLMetadata{
		Hash:    named,
		Address: "https://example.org/somewhere-else",
	})

	if foundDocument.Hash != named {
		t.Fatalf("Hash = %q, want the name the peer reported %q", foundDocument.Hash, named)
	}
}

func TestTheWordsOfAReadPageStandForTheWordsAPeerCounted(t *testing.T) {
	t.Parallel()

	foundDocument := queryanswers.FoundDocument{
		AmountOfWordsAPeerCounted:  yacymodel.Some(1200),
		AmountOfWordsOfTheReadPage: yacymodel.Some(400),
	}

	if foundDocument.AmountOfWordsAnyoneCounted().OrElse(0) != 400 {
		t.Fatalf(
			"the found document holds %+v words anyone counted, want the 400 words of the page "+
				"this node read",
			foundDocument.AmountOfWordsAnyoneCounted(),
		)
	}
}

func TestADocumentWhosePageNoNodeReadHoldsTheWordsAPeerCounted(t *testing.T) {
	t.Parallel()

	foundDocument := queryanswers.FoundDocument{
		AmountOfWordsAPeerCounted: yacymodel.Some(1200),
	}

	if foundDocument.AmountOfWordsAnyoneCounted().OrElse(0) != 1200 {
		t.Fatalf(
			"the found document holds %+v words anyone counted, want the 1200 words the peer "+
				"counted",
			foundDocument.AmountOfWordsAnyoneCounted(),
		)
	}
}

func TestADocumentNobodyCountedTheWordsOfHoldsNoAmountOfWords(t *testing.T) {
	t.Parallel()

	foundDocument := queryanswers.FoundDocument{}

	if foundDocument.AmountOfWordsAnyoneCounted().Present() {
		t.Fatalf(
			"the found document holds %+v words anyone counted, want none",
			foundDocument.AmountOfWordsAnyoneCounted(),
		)
	}
}
