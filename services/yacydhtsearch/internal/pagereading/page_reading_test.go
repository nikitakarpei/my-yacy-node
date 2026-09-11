package pagereading_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	pageReadBudget       = time.Second
	snippetLengthCeiling = 40
	addressOfTheDocument = "https://berlin.example/"
	visibleTextOfThePage = "Berlin Berlin holds a wall."
	pageOfBerlin         = `<!doctype html><html><head><title>Berlin</title>` +
		`<script>var div = "berlin berlin berlin";</script></head>` +
		`<body class="berlin page"><div id="wall">` +
		`<p>Berlin holds a wall.</p></div></body></html>`
)

type pagesHeldAtTheirAddress map[string]string

func (pages pagesHeldAtTheirAddress) Fetch(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	body, held := pages[pageURL.String()]
	if !held {
		return pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}, nil
	}

	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			ContentType: "text/html; charset=utf-8",
			Body:        []byte(body),
		},
	}, nil
}

type pagesThatOutlastTheBudget struct{}

func (pagesThatOutlastTheBudget) Fetch(
	ctx context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	<-ctx.Done()

	//nolint:wrapcheck // the budget that ended is what the reading reports
	return pagefetch.FetchOutcome{}, ctx.Err()
}

type pagesOfAnUnsupportedKind struct{}

func (pagesOfAnUnsupportedKind) Fetch(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			ContentType: "application/octet-stream",
			Body:        []byte("berlin"),
		},
	}, nil
}

type recordedPageReading struct {
	performed pagereading.PerformedPageReading
}

func (r *recordedPageReading) PageReadingPerformed(
	_ context.Context,
	performed pagereading.PerformedPageReading,
) {
	r.performed = performed
}

func readingOfThePages(
	t *testing.T,
	pageFetch pagefetch.Fetcher,
	observer pagereading.PageReadingObserver,
) pagereading.Reading {
	t.Helper()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("pageformats.New(): %v", err)
	}

	return pagereading.New(
		pageFetch,
		formatDerivations,
		pageReadBudget,
		snippetLengthCeiling,
		pagereading.PageReadingObservers{observer},
	)
}

func documentOfTheAddress(t *testing.T) yacymodel.URLHash {
	t.Helper()

	document, err := yacymodel.URLHashOf(addressOfTheDocument)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", addressOfTheDocument, err)
	}

	return document
}

func pageToReadOfTheDocument(t *testing.T) pagereading.PageToRead {
	t.Helper()

	return pagereading.PageToRead{
		Document: documentOfTheAddress(t),
		Address:  addressOfTheDocument,
	}
}

func pagesHoldingTheDocument(t *testing.T) pagesHeldAtTheirAddress {
	t.Helper()

	pageURL, err := canonicalurl.CanonicalURLOf(addressOfTheDocument)
	if err != nil {
		t.Fatalf("CanonicalURLOf(%q): %v", addressOfTheDocument, err)
	}

	return pagesHeldAtTheirAddress{pageURL.String(): pageOfBerlin}
}

func textOfTheDocumentRead(
	t *testing.T,
	pageTextPerDocument map[yacymodel.URLHash]pagereading.PageText,
) pagereading.PageText {
	t.Helper()

	pageText, read := pageTextPerDocument[documentOfTheAddress(t)]
	if !read {
		t.Fatalf("the reading gives %+v, want the text of the page", pageTextPerDocument)
	}

	return pageText
}

func TestThePageOfADocumentGivesTheHitsOfEachQueryWordInItsText(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocument(t), &recordedPageReading{})

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin"), yacymodel.WordHash("wall")},
		[]pagereading.PageToRead{pageToReadOfTheDocument(t)},
	)

	pageText := textOfTheDocumentRead(t, pageTextPerDocument)
	if pageText.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 2 ||
		pageText.HitsPerQueryWord[yacymodel.WordHash("wall")] != 1 {
		t.Fatalf("the page gives %+v, want two hits of berlin and one of wall", pageText)
	}
}

func TestOnlyTheWordsAReaderSeesInThePageAreRead(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocument(t), &recordedPageReading{})

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{
			yacymodel.WordHash("berlin"),
			yacymodel.WordHash("div"),
			yacymodel.WordHash("body"),
		},
		[]pagereading.PageToRead{pageToReadOfTheDocument(t)},
	)

	pageText := textOfTheDocumentRead(t, pageTextPerDocument)
	if pageText.AmountOfWords != len(yacymodel.WordsIn(visibleTextOfThePage)) {
		t.Fatalf(
			"the page holds %d words, want the %d words of %q",
			pageText.AmountOfWords,
			len(yacymodel.WordsIn(visibleTextOfThePage)),
			visibleTextOfThePage,
		)
	}
	if pageText.HitsPerQueryWord[yacymodel.WordHash("div")] != 0 ||
		pageText.HitsPerQueryWord[yacymodel.WordHash("body")] != 0 {
		t.Fatalf("the page gives %+v, want no hit for a word that is markup", pageText)
	}
	if strings.Contains(pageText.Snippet, "<") {
		t.Fatalf("the snippet reads %q, want text without markup", pageText.Snippet)
	}
}

func TestTheSnippetOfADocumentStartsAtTheFirstQueryWordOfItsPage(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocument(t), &recordedPageReading{})

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("holds")},
		[]pagereading.PageToRead{pageToReadOfTheDocument(t)},
	)

	snippet := textOfTheDocumentRead(t, pageTextPerDocument).Snippet
	if !strings.HasPrefix(snippet, "holds a wall") ||
		len([]rune(snippet)) > snippetLengthCeiling {
		t.Fatalf("the snippet reads %q, want the text from the first query word on", snippet)
	}
}

func TestAPageThatNoAddressHoldsIsRefused(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesHeldAtTheirAddress{}, observer)

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheDocument(t)},
	)

	if len(pageTextPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", pageTextPerDocument)
	}
	if observer.performed.AmountOfPagesToRead != 1 ||
		observer.performed.AmountOfPagesRefused != 1 {
		t.Fatalf(
			"PageReadingPerformed = %+v, want one page to read and one refused",
			observer.performed,
		)
	}
}

func TestAPageAtAnAddressThatIsNoWebAddressIsUnreachable(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesHoldingTheDocument(t), observer)

	reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{{Document: documentOfTheAddress(t), Address: "berlin"}},
	)

	if observer.performed.AmountOfPagesUnreachable != 1 {
		t.Fatalf(
			"PageReadingPerformed = %+v, want one unreachable page",
			observer.performed,
		)
	}
}

func TestAPageThatOutlastsTheReadBudgetGivesNothingForItsDocument(t *testing.T) {
	t.Parallel()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("pageformats.New(): %v", err)
	}
	observer := &recordedPageReading{}
	reading := pagereading.New(
		pagesThatOutlastTheBudget{},
		formatDerivations,
		time.Millisecond,
		snippetLengthCeiling,
		pagereading.PageReadingObservers{observer},
	)

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheDocument(t)},
	)

	if len(pageTextPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", pageTextPerDocument)
	}
	if observer.performed.AmountOfPagesOutOfBudget != 1 {
		t.Fatalf("PageReadingPerformed = %+v, want one page out of budget", observer.performed)
	}
}

func TestAPageOfAnUnsupportedKindGivesNothingForItsDocument(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesOfAnUnsupportedKind{}, observer)

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheDocument(t)},
	)

	if len(pageTextPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", pageTextPerDocument)
	}
	if observer.performed.AmountOfPagesOfAnUnsupportedKind != 1 {
		t.Fatalf(
			"PageReadingPerformed = %+v, want one page of an unsupported kind",
			observer.performed,
		)
	}
}
