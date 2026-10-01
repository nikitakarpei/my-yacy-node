package pagereading_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	pageReadBudget              = time.Second
	maxRedirectHops             = 3
	snippetLengthCeiling        = 40
	addressOfTheDocument        = "https://berlin.example/"
	readableTextOfThePage       = "Berlin holds a wall."
	fullTextOfThePage           = "Berlin Berlin holds a wall."
	addressOfTheLinkingDocument = "https://berlin.example/wall"
	pageOfTheLinkingDocument    = `<!doctype html><html><head><title>Berlin</title>` +
		`</head><body><p>Berlin holds a wall.</p>` +
		`<a href="/gate">the gate</a><a href="/wall">the wall</a>` +
		`<a href="https://other.example/berlin">another site</a></body></html>`
	pageOfBerlin = `<!doctype html><html><head><title>Berlin</title>` +
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

type pagesWhoseDeadlinePassed struct{}

func (pagesWhoseDeadlinePassed) Fetch(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return pagefetch.FetchOutcome{Status: pagefetch.FetchDeadlinePassed}, nil
}

type pagesHeldAtTheirAddressAfterAWhile struct {
	pages pagesHeldAtTheirAddress
	while time.Duration
}

func (pages pagesHeldAtTheirAddressAfterAWhile) Fetch(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	time.Sleep(pages.while)

	return pages.pages.Fetch(ctx, pageURL, knownVersion)
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

var cutoffNever = pagereading.PageReadCutoff{PercentOfPages: 100}

type clockThatNeverFires struct{}

func (clockThatNeverFires) After(_ time.Duration, _ func()) func() {
	return func() {}
}

type clockTheTestFires struct {
	graces chan func()
}

func (clock clockTheTestFires) After(_ time.Duration, expire func()) func() {
	clock.graces <- expire

	return func() {}
}

func pagesReadFrom(
	t *testing.T,
	reading pagereading.Reading,
	queryWords []yacymodel.Hash,
	pagesToRead []pagereading.PageToRead,
) pagereading.PagesRead {
	t.Helper()

	run := reading.Start(queryWords)
	run.StartReading(t.Context(), pagesToRead)

	return run.PagesReadAmong(t.Context(), pagesToRead)
}

func readingOfThePages(
	t *testing.T,
	pageFetch pagefetch.Fetcher,
	observer pagereading.PageReadingObserver,
) pagereading.Reading {
	t.Helper()

	return readingCutOffBy(t, pageFetch, observer, cutoffNever, clockThatNeverFires{})
}

func readingCutOffBy(
	t *testing.T,
	pageFetch pagefetch.Fetcher,
	observer pagereading.PageReadingObserver,
	cutoff pagereading.PageReadCutoff,
	clock pagereading.Clock,
) pagereading.Reading {
	t.Helper()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("pageformats.New(): %v", err)
	}

	return pagereading.New(
		redirectfollowingfetch.New(pageFetch, maxRedirectHops),
		formatDerivations,
		pageReadBudget,
		cutoff,
		clock,
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
		addressOfTheDocument:        pageOfBerlin,
		addressOfTheArticle:         pageOfTheArticle,
		addressOfTheLinkingDocument: pageOfTheLinkingDocument,
	} {
		pageURL, err := canonicalurl.CanonicalURLOf(address)
		if err != nil {
			t.Fatalf("CanonicalURLOf(%q): %v", address, err)
		}
		pages[pageURL.String()] = page
	}

	return pages
}

func pageContentsOfTheAddressRead(
	t *testing.T,
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
	address string,
) pagecontents.PageContents {
	t.Helper()

	pageContents, read := pageContentsPerDocument[documentOfTheAddress(t, address)]
	if !read {
		t.Fatalf("the reading gives %+v, want the contents of the page", pageContentsPerDocument)
	}

	return pageContents
}

func TestThePageOfADocumentGivesTheHitsOfEachQueryWordInItsText(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin"), yacymodel.WordHash("wall")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	pageContents := pageContentsOfTheAddressRead(t, pageContentsPerDocument, addressOfTheDocument)
	if pageContents.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 1 ||
		pageContents.HitsPerQueryWord[yacymodel.WordHash("wall")] != 1 {
		t.Fatalf("the page gives %+v, want one hit of berlin and one of wall", pageContents)
	}
}

func TestThePageOfADocumentGivesItsTitle(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	pageContents := pageContentsOfTheAddressRead(t, pageContentsPerDocument, addressOfTheDocument)
	if pageContents.Title != "Berlin" {
		t.Fatalf("the page gives the title %q, want the title the page holds", pageContents.Title)
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

	pageContentsPerDocument := pagesReadFrom(
		t, reading, words, []pagereading.PageToRead{pageToReadOfTheAddress(t, address)},
	).PageContentsPerDocument

	return pageContentsOfTheAddressRead(t, pageContentsPerDocument, address).QueryPhraseHits
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

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{
			yacymodel.WordHash("berlin"),
			yacymodel.WordHash("div"),
			yacymodel.WordHash("body"),
		},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	pageContents := pageContentsOfTheAddressRead(t, pageContentsPerDocument, addressOfTheDocument)
	if pageContents.AmountOfWords != len(yacymodel.WordsIn(readableTextOfThePage)) {
		t.Fatalf(
			"the page holds %d words, want the %d words of %q",
			pageContents.AmountOfWords,
			len(yacymodel.WordsIn(readableTextOfThePage)),
			readableTextOfThePage,
		)
	}
	if pageContents.HitsPerQueryWord[yacymodel.WordHash("div")] != 0 ||
		pageContents.HitsPerQueryWord[yacymodel.WordHash("body")] != 0 {
		t.Fatalf("the page gives %+v, want no hit for a word that is markup", pageContents)
	}
	if strings.Contains(pageContents.Snippet, "<") {
		t.Fatalf("the snippet reads %q, want text without markup", pageContents.Snippet)
	}
}

func TestTheSnippetOfADocumentIsCutAtAWordBoundaryBeforeItsLengthCeiling(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("terraform")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheArticle)},
	).PageContentsPerDocument

	snippet := pageContentsOfTheAddressRead(t, pageContentsPerDocument, addressOfTheArticle).Snippet
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

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{
			yacymodel.WordHash("terraform"),
			yacymodel.WordHash("navigation"),
		},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheArticle)},
	).PageContentsPerDocument

	pageContents := pageContentsOfTheAddressRead(t, pageContentsPerDocument, addressOfTheArticle)
	if pageContents.HitsPerQueryWord[yacymodel.WordHash("navigation")] != 0 {
		t.Fatalf("the page gives %+v, want no hit for a word only the menu holds", pageContents)
	}
	if !strings.HasPrefix(strings.ToLower(pageContents.Snippet), "terraform") {
		t.Fatalf("the snippet reads %q, want the text of the article", pageContents.Snippet)
	}
	for _, wordOfTheNavigation := range yacymodel.WordsIn(navigationOfTheArticle) {
		if strings.Contains(strings.ToLower(pageContents.Snippet), wordOfTheNavigation) {
			t.Fatalf(
				"the snippet reads %q, want no word of the menu %q",
				pageContents.Snippet,
				navigationOfTheArticle,
			)
		}
	}
}

func TestTheWholeTextOfAPageIsReadWhenItHoldsNoReadableText(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := pagereading.New(
		redirectfollowingfetch.New(pagesHoldingTheDocuments(t), maxRedirectHops),
		documentsThatHoldNoReadableText{},
		pageReadBudget,
		cutoffNever,
		clockThatNeverFires{},
		snippetLengthCeiling,
		pagereading.PageReadingObservers{observer},
	)

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	pageContents := pageContentsOfTheAddressRead(t, pageContentsPerDocument, addressOfTheDocument)
	if pageContents.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 2 ||
		pageContents.AmountOfWords != len(yacymodel.WordsIn(fullTextOfThePage)) {
		t.Fatalf("the page gives %+v, want the whole text of the page", pageContents)
	}
}

func TestAPageThatNoAddressHoldsIsRefused(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesHeldAtTheirAddress{}, observer)

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	if len(pageContentsPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", pageContentsPerDocument)
	}
	if observer.performed.AmountOfPagesToRead != 1 ||
		observer.performed.AmountOfPagesRefused != 1 {
		t.Fatalf(
			"PageReadingPerformed = %+v, want one page to read and one refused",
			observer.performed,
		)
	}
}

type pagesGone struct{}

func (pagesGone) Fetch(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return pagefetch.FetchOutcome{Status: pagefetch.FetchGone}, nil
}

func TestAPageTheSiteSaysIsGoneGivesItsDocumentAsWithdrawn(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesGone{}, observer)

	pagesRead := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	if _, withdrawn := pagesRead.WithdrawnDocuments[documentOfTheAddress(t, addressOfTheDocument)]; !withdrawn ||
		len(pagesRead.PageContentsPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want the document as withdrawn and no text", pagesRead)
	}
	if observer.performed.AmountOfPagesGone != 1 {
		t.Fatalf("PageReadingPerformed = %+v, want one gone page", observer.performed)
	}
}

type pagesStatingRobotsRules struct {
	robotsTagValues []string
	body            string
}

func (pages pagesStatingRobotsRules) Fetch(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			ContentType:     "text/html; charset=utf-8",
			Body:            []byte(pages.body),
			RobotsTagValues: pages.robotsTagValues,
		},
	}, nil
}

func TestAPageThatRefusesIndexingGivesItsDocumentAsWithdrawn(t *testing.T) {
	t.Parallel()

	for name, pages := range map[string]pagesStatingRobotsRules{
		"in the header": {
			robotsTagValues: []string{"noindex"},
			body:            pageOfBerlin,
		},
		"in the page": {
			body: strings.Replace(
				pageOfBerlin, "<title>", `<meta name="robots" content="none"><title>`, 1,
			),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			observer := &recordedPageReading{}
			reading := readingOfThePages(t, pages, observer)

			pagesRead := pagesReadFrom(
				t, reading,
				[]yacymodel.Hash{yacymodel.WordHash("berlin")},
				[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
			)

			document := documentOfTheAddress(t, addressOfTheDocument)
			if _, withdrawn := pagesRead.WithdrawnDocuments[document]; !withdrawn ||
				len(pagesRead.PageContentsPerDocument) != 0 {
				t.Fatalf(
					"the reading gives %+v, want the document as withdrawn and no text",
					pagesRead,
				)
			}
			if observer.performed.AmountOfPagesRefusingIndexing != 1 {
				t.Fatalf(
					"PageReadingPerformed = %+v, want one page refusing indexing",
					observer.performed,
				)
			}
		})
	}
}

func TestAPageThatAllowsIndexingIsRead(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesStatingRobotsRules{
		robotsTagValues: []string{"nofollow", "googlebot: noindex"},
		body:            pageOfBerlin,
	}, &recordedPageReading{})

	pagesRead := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	if len(pagesRead.WithdrawnDocuments) != 0 || len(pagesRead.PageContentsPerDocument) != 1 {
		t.Fatalf("the reading gives %+v, want the page read", pagesRead)
	}
}

type pagesAssessedForSpam struct {
	spamAssessmentValue string
}

func (pages pagesAssessedForSpam) Fetch(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			ContentType:         "text/html; charset=utf-8",
			Body:                []byte(pageOfBerlin),
			SpamAssessmentValue: pages.spamAssessmentValue,
		},
	}, nil
}

func TestAReadPageGivesTheSpamVerdictOfItsHeader(t *testing.T) {
	t.Parallel()

	for spamAssessmentValue, wantedVerdict := range map[string]spamassessment.Verdict{
		`spam;score=0.935;threshold=0.8;model="2026-09"`:  spamassessment.Spam,
		`clean;score=0.007;threshold=0.8;model="2026-09"`: spamassessment.Clean,
		"": spamassessment.Unassessed,
	} {
		t.Run(spamAssessmentValue, func(t *testing.T) {
			t.Parallel()

			observer := &recordedPageReading{}
			reading := readingOfThePages(t, pagesAssessedForSpam{spamAssessmentValue}, observer)

			pagesRead := pagesReadFrom(
				t, reading,
				[]yacymodel.Hash{yacymodel.WordHash("berlin")},
				[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
			)

			document := documentOfTheAddress(t, addressOfTheDocument)
			if verdict := pagesRead.SpamVerdictPerDocument[document]; verdict != wantedVerdict {
				t.Fatalf("the reading gives the verdict %v, want %v", verdict, wantedVerdict)
			}
			if observer.performed.AmountOfPagesReadPerSpamVerdict[wantedVerdict] != 1 {
				t.Fatalf(
					"PageReadingPerformed = %+v, want one read page of the verdict %v",
					observer.performed, wantedVerdict,
				)
			}
		})
	}
}

func TestAPageTheSiteRefusesIsNotWithdrawn(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHeldAtTheirAddress{}, &recordedPageReading{})

	pagesRead := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	if len(pagesRead.WithdrawnDocuments) != 0 {
		t.Fatalf("the reading gives %+v, want no withdrawn document", pagesRead)
	}
}

func TestAPageTooLargeToFetchIsRefused(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesTooLargeToFetch{}, observer)

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	if len(pageContentsPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", pageContentsPerDocument)
	}
	if observer.performed.AmountOfPagesRefused != 1 {
		t.Fatalf("PageReadingPerformed = %+v, want one refused page", observer.performed)
	}
}

func TestAPageAtAnAddressThatIsNoWebAddressIsUnreachable(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), observer)

	pagesReadFrom(
		t, reading,
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
		redirectfollowingfetch.New(pagesThatOutlastTheBudget{}, maxRedirectHops),
		formatDerivations,
		time.Millisecond,
		cutoffNever,
		clockThatNeverFires{},
		snippetLengthCeiling,
		pagereading.PageReadingObservers{observer},
	)

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	if len(pageContentsPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", pageContentsPerDocument)
	}
	if observer.performed.AmountOfPagesOutOfBudget != 1 {
		t.Fatalf("PageReadingPerformed = %+v, want one page out of budget", observer.performed)
	}
}

func TestAPageOfAnUnsupportedKindGivesNothingForItsDocument(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesOfAnUnsupportedKind{}, observer)

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	if len(pageContentsPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", pageContentsPerDocument)
	}
	if observer.performed.AmountOfPagesOfAnUnsupportedKind != 1 {
		t.Fatalf(
			"PageReadingPerformed = %+v, want one page of an unsupported kind",
			observer.performed,
		)
	}
}

func TestAPageWhoseFetchDeadlinePassedIsOutOfBudget(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesWhoseDeadlinePassed{}, observer)

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	if len(pageContentsPerDocument) != 0 {
		t.Fatalf("the reading gives %+v, want nothing for the document", pageContentsPerDocument)
	}
	if observer.performed.AmountOfPagesOutOfBudget != 1 {
		t.Fatalf("PageReadingPerformed = %+v, want one page out of budget", observer.performed)
	}
}

func TestThePageReadingTellsTheTimeItSpentFetchingApartFromReading(t *testing.T) {
	t.Parallel()

	timeToFetchThePage := 20 * time.Millisecond
	observer := &recordedPageReading{}
	reading := readingOfThePages(
		t,
		pagesHeldAtTheirAddressAfterAWhile{
			pages: pagesHoldingTheDocuments(t),
			while: timeToFetchThePage,
		},
		observer,
	)

	pagesReadFrom(
		t,
		reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	)

	performed := observer.performed
	if performed.TimeSpentFetching < timeToFetchThePage {
		t.Fatalf(
			"PageReadingPerformed = %+v, want at least %v spent fetching",
			performed, timeToFetchThePage,
		)
	}
	if performed.TimeSpentReading <= 0 {
		t.Fatalf("PageReadingPerformed = %+v, want time spent reading", performed)
	}
}

type pagesThatOutlastTheBudgetAtOneAddress struct {
	pages       pagesHeldAtTheirAddress
	slowAddress string
}

func (p pagesThatOutlastTheBudgetAtOneAddress) Fetch(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	if pageURL.String() == p.slowAddress {
		return pagesThatOutlastTheBudget{}.Fetch(ctx, pageURL, knownVersion)
	}

	return p.pages.Fetch(ctx, pageURL, knownVersion)
}

func pagesOfThreeDocuments(t *testing.T) []pagereading.PageToRead {
	t.Helper()

	return []pagereading.PageToRead{
		pageToReadOfTheAddress(t, addressOfTheDocument),
		pageToReadOfTheAddress(t, addressOfTheLinkingDocument),
		pageToReadOfTheAddress(t, addressOfTheArticle),
	}
}

func pagesThatOutlastTheBudgetAtTheArticle(t *testing.T) pagesThatOutlastTheBudgetAtOneAddress {
	t.Helper()

	return pagesThatOutlastTheBudgetAtOneAddress{
		pages:       pagesHoldingTheDocuments(t),
		slowAddress: addressOfTheArticle,
	}
}

func TestAPageStillBeingReadAfterTheGraceIsCutOff(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	clock := clockTheTestFires{graces: make(chan func(), 1)}
	reading := readingCutOffBy(
		t,
		pagesThatOutlastTheBudgetAtTheArticle(t),
		observer,
		pagereading.PageReadCutoff{PercentOfPages: 50},
		clock,
	)
	run := reading.Start([]yacymodel.Hash{yacymodel.WordHash("berlin")})
	run.StartReading(t.Context(), pagesOfThreeDocuments(t))

	pagesRead := make(chan pagereading.PagesRead, 1)
	go func() { pagesRead <- run.PagesReadAmong(t.Context(), pagesOfThreeDocuments(t)) }()
	expireTheGrace := <-clock.graces
	expireTheGrace()
	<-pagesRead

	performed := observer.performed
	if performed.AmountOfPagesRead != 2 || performed.AmountOfPagesCutOff != 1 ||
		performed.AmountOfPagesOutOfBudget != 0 {
		t.Fatalf(
			"PageReadingPerformed = %+v, want two pages read and the slow page cut off",
			performed,
		)
	}
}

func TestAPageStillBeingReadIsOutOfBudgetWhenTheCutoffIsOff(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesThatOutlastTheBudgetAtTheArticle(t), observer)

	pagesReadFrom(
		t, reading, []yacymodel.Hash{yacymodel.WordHash("berlin")}, pagesOfThreeDocuments(t),
	)

	performed := observer.performed
	if performed.AmountOfPagesRead != 2 || performed.AmountOfPagesOutOfBudget != 1 ||
		performed.AmountOfPagesCutOff != 0 || performed.TimeSpent < pageReadBudget {
		t.Fatalf(
			"PageReadingPerformed = %+v, want two pages read and the slow page out of budget",
			performed,
		)
	}
}

type pagesCountingTheirFetches struct {
	pages           pagesHeldAtTheirAddress
	amountOfFetches *atomic.Int32
}

func (p pagesCountingTheirFetches) Fetch(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	p.amountOfFetches.Add(1)

	return p.pages.Fetch(ctx, pageURL, knownVersion)
}

func TestAPageReadBeforeItIsWantedIsReadOnce(t *testing.T) {
	t.Parallel()

	amountOfFetches := &atomic.Int32{}
	reading := readingOfThePages(
		t,
		pagesCountingTheirFetches{
			pages:           pagesHoldingTheDocuments(t),
			amountOfFetches: amountOfFetches,
		},
		&recordedPageReading{},
	)
	pagesToRead := []pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)}
	run := reading.Start([]yacymodel.Hash{yacymodel.WordHash("berlin")})
	run.StartReading(t.Context(), pagesToRead)
	run.StartReading(t.Context(), pagesToRead)

	pageContentsPerDocument := run.PagesReadAmong(t.Context(), pagesToRead).PageContentsPerDocument

	pageContentsOfTheAddressRead(t, pageContentsPerDocument, addressOfTheDocument)
	if amountOfFetches.Load() != 1 {
		t.Fatalf("the page was fetched %d times, want once", amountOfFetches.Load())
	}
}

func TestAPageReadButNeverWantedIsLeftOut(t *testing.T) {
	t.Parallel()

	observer := &recordedPageReading{}
	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), observer)
	run := reading.Start([]yacymodel.Hash{yacymodel.WordHash("berlin")})
	run.StartReading(t.Context(), pagesOfThreeDocuments(t))

	pageContentsPerDocument := run.PagesReadAmong(
		t.Context(),
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	pageContentsOfTheAddressRead(t, pageContentsPerDocument, addressOfTheDocument)
	if len(pageContentsPerDocument) != 1 {
		t.Fatalf("the reading gives %+v, want only the page wanted", pageContentsPerDocument)
	}
	if observer.performed.AmountOfPagesToRead != 1 ||
		observer.performed.AmountOfPagesUnwanted != 2 {
		t.Fatalf(
			"PageReadingPerformed = %+v, want one page to read and two unwanted",
			observer.performed,
		)
	}
}

func TestAPageWantedButNeverStartedIsOutOfBudgetUnread(t *testing.T) {
	t.Parallel()

	amountOfFetches := &atomic.Int32{}
	observer := &recordedPageReading{}
	reading := readingOfThePages(
		t,
		pagesCountingTheirFetches{
			pages:           pagesHoldingTheDocuments(t),
			amountOfFetches: amountOfFetches,
		},
		observer,
	)
	run := reading.Start([]yacymodel.Hash{yacymodel.WordHash("berlin")})

	pageContentsPerDocument := run.PagesReadAmong(
		t.Context(),
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	if len(pageContentsPerDocument) != 0 || amountOfFetches.Load() != 0 {
		t.Fatalf(
			"the reading gives %+v after %d fetches, want no page and no fetch",
			pageContentsPerDocument, amountOfFetches.Load(),
		)
	}
	if observer.performed.AmountOfPagesOutOfBudget != 1 {
		t.Fatalf("PageReadingPerformed = %+v, want one page out of budget", observer.performed)
	}
}

type pagesMovedToTheDocument struct {
	pages      pagesHeldAtTheirAddress
	movedPages map[string]canonicalurl.CanonicalURL
}

func (p pagesMovedToTheDocument) Fetch(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	if target, moved := p.movedPages[pageURL.String()]; moved {
		return pagefetch.FetchOutcome{
			Status:         pagefetch.FetchRedirected,
			RedirectTarget: target,
		}, nil
	}

	return p.pages.Fetch(ctx, pageURL, knownVersion)
}

func pagesMovedFrom(t *testing.T, movedAddress string) pagesMovedToTheDocument {
	t.Helper()

	target, err := canonicalurl.CanonicalURLOf(addressOfTheDocument)
	if err != nil {
		t.Fatalf("CanonicalURLOf(%q): %v", addressOfTheDocument, err)
	}
	moved, err := canonicalurl.CanonicalURLOf(movedAddress)
	if err != nil {
		t.Fatalf("CanonicalURLOf(%q): %v", movedAddress, err)
	}

	return pagesMovedToTheDocument{
		pages:      pagesHoldingTheDocuments(t),
		movedPages: map[string]canonicalurl.CanonicalURL{moved.String(): target},
	}
}

func TestAPageThatMovedIsReadAtTheAddressItMovedTo(t *testing.T) {
	t.Parallel()

	movedAddress := "https://old.example/berlin"
	reading := readingOfThePages(t, pagesMovedFrom(t, movedAddress), &recordedPageReading{})

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, movedAddress)},
	).PageContentsPerDocument

	pageContents := pageContentsOfTheAddressRead(t, pageContentsPerDocument, movedAddress)
	if pageContents.Title != "Berlin" || pageContents.Address != addressOfTheDocument {
		t.Fatalf(
			"the page gives %+v, want the text of %s and its address",
			pageContents, addressOfTheDocument,
		)
	}
}

func TestAPageThatDidNotMoveGivesNoAddress(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheDocument)},
	).PageContentsPerDocument

	pageContents := pageContentsOfTheAddressRead(t, pageContentsPerDocument, addressOfTheDocument)
	if pageContents.Address != "" {
		t.Fatalf("the page gives the address %q, want none for a page that did not move",
			pageContents.Address)
	}
}

func TestThePageOfADocumentGivesTheLinksOfItsOwnSiteAndOfOtherSitesItHolds(t *testing.T) {
	t.Parallel()

	reading := readingOfThePages(t, pagesHoldingTheDocuments(t), &recordedPageReading{})

	pageContentsPerDocument := pagesReadFrom(
		t, reading,
		[]yacymodel.Hash{yacymodel.WordHash("berlin")},
		[]pagereading.PageToRead{pageToReadOfTheAddress(t, addressOfTheLinkingDocument)},
	).PageContentsPerDocument

	linkCounts := pageContentsOfTheAddressRead(
		t, pageContentsPerDocument, addressOfTheLinkingDocument,
	).LinkCounts
	if linkCounts.LocalLinks != 2 || linkCounts.ExternalLinks != 1 {
		t.Fatalf(
			"the page gives the link counts %+v, want the 2 links of its own site and the 1 link "+
				"of another site it holds",
			linkCounts,
		)
	}
}
