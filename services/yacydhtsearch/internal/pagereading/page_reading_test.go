package pagereading_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	pageReadBudget        = time.Second
	snippetLengthCeiling  = 40
	addressOfTheDocument  = "https://berlin.example/"
	readableTextOfThePage = "Berlin holds a wall."
	fullTextOfThePage     = "Berlin Berlin holds a wall."
	pageOfBerlin          = `<!doctype html><html><head><title>Berlin</title>` +
		`<script>var div = "berlin berlin berlin";</script></head>` +
		`<body class="berlin page"><div id="wall">` +
		`<p>Berlin holds a wall.</p></div></body></html>`
)

const (
	addressOfTheArticle    = "https://terraform.example/"
	navigationOfTheArticle = "Skip to content Navigation Menu Sign in"
	paragraphOfTheArticle  = "Terraform writes the state of the infrastructure " +
		"to a file, and the team reads that file to learn what the cloud holds " +
		"today and what the next change of the cloud must do."
	pageOfTheArticle = `<!doctype html><html><head><title>Terraform</title></head>` +
		`<body><nav>` + navigationOfTheArticle + `</nav>` +
		`<article><p>` + paragraphOfTheArticle + `</p>` +
		`<p>` + paragraphOfTheArticle + `</p></article></body></html>`
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

type documentsThatHoldNoReadableText struct{}

func (documentsThatHoldNoReadableText) BodyIn(
	_ context.Context,
	format documentextraction.Format,
	_ documentextraction.Document,
	_ canonicalurl.CanonicalURL,
) ([]byte, bool) {
	if format == documentextraction.FormatReadableText {
		return []byte("  \n  "), true
	}

	return []byte(fullTextOfThePage), true
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

func documentOfTheAddress(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	document, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return document
}

func pageToReadOfTheAddress(t *testing.T, address string) pagereading.PageToRead {
	t.Helper()

	return pagereading.PageToRead{
		Document: documentOfTheAddress(t, address),
		Address:  address,
	}
}

func pagesHoldingTheDocuments(t *testing.T) pagesHeldAtTheirAddress {
	t.Helper()

	pages := pagesHeldAtTheirAddress{}
	for address, page := range map[string]string{
		addressOfTheDocument: pageOfBerlin,
		addressOfTheArticle:  pageOfTheArticle,
	} {
		pageURL, err := canonicalurl.CanonicalURLOf(address)
		if err != nil {
			t.Fatalf("CanonicalURLOf(%q): %v", address, err)
		}
		pages[pageURL.String()] = page
	}

	return pages
}

func textOfTheAddressRead(
	t *testing.T,
	pageTextPerDocument map[yacymodel.URLHash]pagereading.PageText,
	address string,
) pagereading.PageText {
	t.Helper()

	pageText, read := pageTextPerDocument[documentOfTheAddress(t, address)]
	if !read {
		t.Fatalf("the reading gives %+v, want the text of the page", pageTextPerDocument)
	}

	return pageText
}

func TestThePageOfADocumentGivesTheHitsOfEachQueryWordInItsText(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin"), yacymodel.WordHash("wall")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	pageText := textOfTheAddressRead(t, pageTextPerDocument, addressOfTheDocument)
	if pageText.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 1 ||
		pageText.HitsPerQueryWord[yacymodel.WordHash("wall")] != 1 {
		t.Fatalf("the page gives %+v, want one hit of berlin and one of wall", pageText)
	}
}

func TestOnlyTheWordsAReaderSeesInThePageAreRead(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{
			yacymodel.WordHash("berlin"),
			yacymodel.WordHash("div"),
			yacymodel.WordHash("body"),
		},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	pageText := textOfTheAddressRead(t, pageTextPerDocument, addressOfTheDocument)
	if pageText.AmountOfWords != len(yacymodel.WordsIn(readableTextOfThePage)) {
		t.Fatalf(
			"the page holds %d words, want the %d words of %q",
			pageText.AmountOfWords,
			len(yacymodel.WordsIn(readableTextOfThePage)),
			readableTextOfThePage,
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

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("holds")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	snippet := textOfTheAddressRead(t, pageTextPerDocument, addressOfTheDocument).Snippet
	if !strings.HasPrefix(snippet, "holds a wall") ||
		len([]rune(snippet)) > snippetLengthCeiling {
		t.Fatalf("the snippet reads %q, want the text from the first query word on", snippet)
	}
}

func TestOnlyTheArticleOfAPageIsReadWhenThePageHoldsOne(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{
			yacymodel.WordHash("terraform"),
			yacymodel.WordHash("navigation"),
		},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheArticle)},
	)

	pageText := textOfTheAddressRead(t, pageTextPerDocument, addressOfTheArticle)
	if pageText.HitsPerQueryWord[yacymodel.WordHash("navigation")] != 0 {
		t.Fatalf("the page gives %+v, want no hit for a word only the menu holds", pageText)
	}
	if !strings.HasPrefix(strings.ToLower(pageText.Snippet), "terraform") {
		t.Fatalf("the snippet reads %q, want the text of the article", pageText.Snippet)
	}
	for _, wordOfTheNavigation := range yacymodel.WordsIn(navigationOfTheArticle) {
		if strings.Contains(strings.ToLower(pageText.Snippet), wordOfTheNavigation) {
			t.Fatalf(
				"the snippet reads %q, want no word of the menu %q",
				pageText.Snippet,
				navigationOfTheArticle,
			)
		}
	}
}

func TestTheWholeTextOfAPageIsReadWhenItHoldsNoReadableText(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := pagereading.New(
		pagesHoldingTheDocuments(t),
		documentsThatHoldNoReadableText{},
		pageReadBudget,
		snippetLengthCeiling,
		pagereading.PageReadingObservers{observer},
	)

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	pageText := textOfTheAddressRead(t, pageTextPerDocument, addressOfTheDocument)
	if pageText.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 2 ||
		pageText.AmountOfWords != len(yacymodel.WordsIn(fullTextOfThePage)) {
		t.Fatalf("the page gives %+v, want the whole text of the page", pageText)
	}
}

func TestAPageThatNoAddressHoldsIsRefused(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesHeldAtTheirAddress{}, observer)

	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
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
	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), observer)

	reading.PageTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{
			{Document: documentOfTheAddress(t, addressOfTheDocument), Address: "berlin"},
		},
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
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
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
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
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
