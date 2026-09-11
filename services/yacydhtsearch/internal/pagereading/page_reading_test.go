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
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
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

type pagesTooLargeToFetch struct{}

func (pagesTooLargeToFetch) Fetch(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return pagefetch.FetchOutcome{Status: pagefetch.FetchOversized}, nil
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

func documentTextOfTheAddressRead(
	t *testing.T,
	documentTextPerDocument map[yacymodel.URLHash]documenttext.DocumentText,
	address string,
) documenttext.DocumentText {
	t.Helper()

	documentText, read := documentTextPerDocument[documentOfTheAddress(t, address)]
	if !read {
		t.Fatalf("the reading gives %+v, want the text of the page", documentTextPerDocument)
	}

	return documentText
}

func TestThePageOfADocumentGivesTheHitsOfEachQueryWordInItsText(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin"), yacymodel.WordHash("wall")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	documentText := documentTextOfTheAddressRead(t, documentTextPerDocument, addressOfTheDocument)
	if documentText.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 1 ||
		documentText.HitsPerQueryWord[yacymodel.WordHash("wall")] != 1 {
		t.Fatalf("the page gives %+v, want one hit of berlin and one of wall", documentText)
	}
}

func queryPhraseHitsOfTheAddressRead(
	t *testing.T,
	address string,
	queryWords ...string,
) int {
	t.Helper()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})
	words := make([]yacymodel.Hash, 0, len(queryWords))
	for _, queryWord := range queryWords {
		words = append(words, yacymodel.WordHash(queryWord))
	}

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(), words, []pagereading.PageToRead{pageToReadOfTheAddress(t, address)},
	)

	return documentTextOfTheAddressRead(t, documentTextPerDocument, address).QueryPhraseHits
}

func TestThePageOfADocumentGivesTheHitsOfEachQueryPhraseInItsText(t *testing.T) {
	t.Parallel()

	queryPhraseHits := queryPhraseHitsOfTheAddressRead(
		t, addressOfTheArticle, "terraform", "writes",
	)

	if queryPhraseHits != 2 {
		t.Fatalf(
			"the page gives %d query phrase hits, want the two paragraphs that hold the phrase",
			queryPhraseHits,
		)
	}
}

func TestAQueryPhraseTheTextHoldsInTheOtherOrderIsNoHit(t *testing.T) {
	t.Parallel()

	queryPhraseHits := queryPhraseHitsOfTheAddressRead(
		t, addressOfTheDocument, "holds", "berlin",
	)

	if queryPhraseHits != 0 {
		t.Fatalf(
			"the page gives %d query phrase hits, want none for the words in the other order",
			queryPhraseHits,
		)
	}
}

func TestAQueryOfOneWordHoldsNoQueryPhrase(t *testing.T) {
	t.Parallel()

	queryPhraseHits := queryPhraseHitsOfTheAddressRead(
		t, addressOfTheDocument, "berlin",
	)

	if queryPhraseHits != 0 {
		t.Fatalf(
			"the page gives %d query phrase hits, want none for a query of one word",
			queryPhraseHits,
		)
	}
}

func TestOnlyTheWordsAReaderSeesInThePageAreRead(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{
			yacymodel.WordHash("berlin"),
			yacymodel.WordHash("div"),
			yacymodel.WordHash("body"),
		},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	documentText := documentTextOfTheAddressRead(t, documentTextPerDocument, addressOfTheDocument)
	if documentText.AmountOfWords != len(yacymodel.WordsIn(readableTextOfThePage)) {
		t.Fatalf(
			"the page holds %d words, want the %d words of %q",
			documentText.AmountOfWords,
			len(yacymodel.WordsIn(readableTextOfThePage)),
			readableTextOfThePage,
		)
	}
	if documentText.HitsPerQueryWord[yacymodel.WordHash("div")] != 0 ||
		documentText.HitsPerQueryWord[yacymodel.WordHash("body")] != 0 {
		t.Fatalf("the page gives %+v, want no hit for a word that is markup", documentText)
	}
	if strings.Contains(documentText.Snippet, "<") {
		t.Fatalf("the snippet reads %q, want text without markup", documentText.Snippet)
	}
}

func TestTheSnippetOfADocumentStartsAtTheFirstQueryWordOfItsPage(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("holds")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	snippet := documentTextOfTheAddressRead(
		t,
		documentTextPerDocument,
		addressOfTheDocument,
	).Snippet
	if !strings.HasPrefix(snippet, "holds a wall") ||
		len([]rune(snippet)) > snippetLengthCeiling {
		t.Fatalf("the snippet reads %q, want the text from the first query word on", snippet)
	}
}

func TestTheSnippetOfADocumentIsCutAtAWordBoundaryBeforeItsLengthCeiling(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("terraform")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheArticle)},
	)

	snippet := documentTextOfTheAddressRead(t, documentTextPerDocument, addressOfTheArticle).Snippet
	if len([]rune(snippet)) > snippetLengthCeiling ||
		!strings.HasPrefix(paragraphOfTheArticle, snippet+" ") {
		t.Fatalf(
			"the snippet reads %q, want at most %d letters of %q ending at a word boundary",
			snippet, snippetLengthCeiling, paragraphOfTheArticle,
		)
	}
}

func TestOnlyTheArticleOfAPageIsReadWhenThePageHoldsOne(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{
			yacymodel.WordHash("terraform"),
			yacymodel.WordHash("navigation"),
		},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheArticle)},
	)

	documentText := documentTextOfTheAddressRead(t, documentTextPerDocument, addressOfTheArticle)
	if documentText.HitsPerQueryWord[yacymodel.WordHash("navigation")] != 0 {
		t.Fatalf("the page gives %+v, want no hit for a word only the menu holds", documentText)
	}
	if !strings.HasPrefix(strings.ToLower(documentText.Snippet), "terraform") {
		t.Fatalf("the snippet reads %q, want the text of the article", documentText.Snippet)
	}
	for _, wordOfTheNavigation := range yacymodel.WordsIn(navigationOfTheArticle) {
		if strings.Contains(strings.ToLower(documentText.Snippet), wordOfTheNavigation) {
			t.Fatalf(
				"the snippet reads %q, want no word of the menu %q",
				documentText.Snippet,
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

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	documentText := documentTextOfTheAddressRead(t, documentTextPerDocument, addressOfTheDocument)
	if documentText.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 2 ||
		documentText.AmountOfWords != len(yacymodel.WordsIn(fullTextOfThePage)) {
		t.Fatalf("the page gives %+v, want the whole text of the page", documentText)
	}
}

func TestAPageThatNoAddressHoldsIsRefused(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesHeldAtTheirAddress{}, observer)

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	if len(documentTextPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", documentTextPerDocument)
	}
	if observer.performed.AmountOfPagesToRead != 1 ||
		observer.performed.AmountOfPagesRefused != 1 {
		t.Fatalf(
			"PageReadingPerformed = %+v, want one page to read and one refused",
			observer.performed,
		)
	}
}

func TestAPageTooLargeToFetchIsRefused(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesTooLargeToFetch{}, observer)

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	if len(documentTextPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", documentTextPerDocument)
	}
	if observer.performed.AmountOfPagesRefused != 1 {
		t.Fatalf("PageReadingPerformed = %+v, want one refused page", observer.performed)
	}
}

func TestAPageAtAnAddressThatIsNoWebAddressIsUnreachable(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), observer)

	reading.DocumentTextPerDocument(
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

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	if len(documentTextPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", documentTextPerDocument)
	}
	if observer.performed.AmountOfPagesOutOfBudget != 1 {
		t.Fatalf("PageReadingPerformed = %+v, want one page out of budget", observer.performed)
	}
}

func TestAPageOfAnUnsupportedKindGivesNothingForItsDocument(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesOfAnUnsupportedKind{}, observer)

	documentTextPerDocument := reading.DocumentTextPerDocument(
		t.Context(),
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	if len(documentTextPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", documentTextPerDocument)
	}
	if observer.performed.AmountOfPagesOfAnUnsupportedKind != 1 {
		t.Fatalf(
			"PageReadingPerformed = %+v, want one page of an unsupported kind",
			observer.performed,
		)
	}
}
