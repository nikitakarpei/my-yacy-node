package pagerwi_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
)

const (
	pageURL       = "http://example.com/article"
	sampleText    = "the quick brown fox the fox"
	documentTitle = "Hello World"
)

var reachedAt = time.Unix(1_700_000_000, 0)

func scrapedPage(t *testing.T, text string) pagescrapecontract.OfferedPage {
	t.Helper()
	return pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, pageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, pageURL),
		ContentType: "text/html",
		Body:        []byte("<html><body>" + text + "</body></html>"),
	}
}

func extractedDocument() documentextraction.Document {
	return documentextraction.Document{
		Title:         documentTitle,
		Format:        documentextraction.FormatDocumentHTML,
		Language:      "en",
		LocalLinks:    3,
		ExternalLinks: 1,
	}
}

func indexOf(t *testing.T, text string) pagerwi.PageRWI {
	t.Helper()
	return pagerwi.Of(scrapedPage(t, text), extractedDocument(), []byte(text), reachedAt)
}

func TestOfIndexesUnderThePageURLRatherThanTheURLTheFetchLandedOn(t *testing.T) {
	const replayURL = "http://archive.example/replay/http://example.com/"
	scraped := scrapedPage(t, sampleText)
	scraped.LandedURL = canonicalurltest.CanonicalURLOf(t, replayURL)

	index := pagerwi.Of(scraped, extractedDocument(), []byte(sampleText), reachedAt)

	if index.PageURL.String() != pageURL {
		t.Fatalf("page url = %q, want %q", index.PageURL, pageURL)
	}
	if index.Metadata.Address != pageURL {
		t.Fatalf("metadata address = %q, want %q", index.Metadata.Address, pageURL)
	}
}

func TestOfProducesPostingsCarryingTheURLHash(t *testing.T) {
	index := indexOf(t, sampleText)

	if len(index.Postings) == 0 {
		t.Fatal("no postings")
	}
	urlHash := urlhashtest.URLHashOf(t, pageURL)
	for _, posting := range index.Postings {
		if posting.URLHash != urlHash {
			t.Fatalf("posting url hash = %q, want %q", posting.URLHash, urlHash)
		}
		if posting.DocumentType != yacymodel.DocumentTypeText {
			t.Fatalf("posting document type = %v, want text", posting.DocumentType)
		}
	}
}

func TestOfCountsRepeatedWords(t *testing.T) {
	index := indexOf(t, sampleText)

	foxHash := yacymodel.WordHash("fox")
	var found bool
	for _, posting := range index.Postings {
		if posting.WordHash == foxHash {
			found = true
			if posting.Hits != 2 {
				t.Fatalf("fox hit count = %d, want 2", posting.Hits)
			}
		}
	}
	if !found {
		t.Fatal("word 'fox' not in postings")
	}
}

func TestOfCarriesTextStatsAndPageReferenceIntoMetadata(t *testing.T) {
	document := extractedDocument()
	metadata := indexOf(t, sampleText).Metadata

	if want := urlhashtest.URLHashOf(t, pageURL); metadata.Hash != want {
		t.Fatalf("metadata url hash = %q, want %q", metadata.Hash, want)
	}
	if metadata.Title != document.Title {
		t.Fatalf("title = %q", metadata.Title)
	}
	if metadata.WordCount == 0 {
		t.Fatal("word count = 0, want nonzero")
	}
	if int(metadata.LocalLinks) != document.LocalLinks ||
		int(metadata.ExternalLinks) != document.ExternalLinks {
		t.Fatalf("links = %d local, %d external", metadata.LocalLinks, metadata.ExternalLinks)
	}
	if loaded, ok := metadata.Loaded.Get(); !ok || loaded != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("loaded = %+v", metadata.Loaded)
	}
}

func TestOfRecordsTheUndeclaredLanguageInEveryPosting(t *testing.T) {
	document := extractedDocument()
	document.Language = ""

	index := pagerwi.Of(scrapedPage(t, sampleText), document, []byte(sampleText), reachedAt)

	for _, posting := range index.Postings {
		if posting.Language != yacymodel.LanguageOfUndeclaredDocument {
			t.Fatalf("language = %q, want %q",
				posting.Language, yacymodel.LanguageOfUndeclaredDocument)
		}
	}
}

func TestOfLeavesMetadataLanguageAbsentWhenThePageDeclaresNone(t *testing.T) {
	document := extractedDocument()
	document.Language = ""

	index := pagerwi.Of(scrapedPage(t, sampleText), document, []byte(sampleText), reachedAt)

	if language, ok := index.Metadata.Language.Get(); ok {
		t.Fatalf("metadata language = %q, want absent", language)
	}
}

func TestOfDropsWordsShorterThanTwoCharacters(t *testing.T) {
	index := indexOf(t, "a fox I saw")

	for _, posting := range index.Postings {
		if posting.WordHash == yacymodel.WordHash("a") ||
			posting.WordHash == yacymodel.WordHash("i") {
			t.Fatalf("short word should not be indexed: %q", posting.WordHash)
		}
	}
	if len(index.Postings) != 2 {
		t.Fatalf("want postings for 'fox' and 'saw' only, got %d", len(index.Postings))
	}
}

func TestOfKeepsHyphenatedCompoundAsOneWord(t *testing.T) {
	assertWordIndexed(t, "state-of-the-art design", "state-of-the-art")
}

func TestOfKeepsDigitSeparatedNumberAsOneWord(t *testing.T) {
	assertWordIndexed(t, "the price is 1,234.56 today", "1,234.56")
}

func TestOfIndexesEveryWordOfThePageText(t *testing.T) {
	words := []string{"navigation", "menu", "the", "quick", "fox"}

	index := indexOf(t, strings.Join(words, " "))

	if len(index.Postings) != len(words) {
		t.Fatalf("postings = %d, want one per word of %v", len(index.Postings), words)
	}
	for _, word := range words {
		assertWordIndexed(t, strings.Join(words, " "), word)
	}
}

func TestOfMetadataByteSizeReflectsTheFetchedDocument(t *testing.T) {
	page := scrapedPage(t, "the quick fox")

	index := pagerwi.Of(page, extractedDocument(), []byte("the quick fox"), reachedAt)

	if size, known := index.Metadata.ByteSize.Get(); !known || int(size) != len(page.Body) {
		t.Fatalf(
			"byte size = %v, want the fetched document size %d",
			index.Metadata.ByteSize,
			len(page.Body),
		)
	}
}

func TestOfPostingsMeasureThePageURL(t *testing.T) {
	index := indexOf(t, sampleText)

	for _, posting := range index.Postings {
		if int(posting.URLLength) != len(pageURL) {
			t.Fatalf("url length = %d, want %d", posting.URLLength, len(pageURL))
		}
		if posting.URLComponents != 3 {
			t.Fatalf("url components = %d, want 3", posting.URLComponents)
		}
	}
}

func TestOfCountsPhrasesAndPhrasePositions(t *testing.T) {
	index := indexOf(t, "the quick fox jumps. the lazy dog sleeps.")

	for _, posting := range index.Postings {
		if posting.Phrases != 2 {
			t.Fatalf("phrase count = %d, want 2", posting.Phrases)
		}
	}
	jumpsHash := yacymodel.WordHash("jumps")
	sleepsHash := yacymodel.WordHash("sleeps")
	var jumpsPhrase, sleepsPhrase uint8
	var jumpsFound, sleepsFound bool
	for _, posting := range index.Postings {
		if posting.WordHash == jumpsHash {
			jumpsPhrase, jumpsFound = posting.PhrasePosition, true
		}
		if posting.WordHash == sleepsHash {
			sleepsPhrase, sleepsFound = posting.PhrasePosition, true
		}
	}
	if !jumpsFound || !sleepsFound || jumpsPhrase == sleepsPhrase {
		t.Fatalf(
			"words in different sentences should get different phrase numbers, got %d and %d",
			jumpsPhrase,
			sleepsPhrase,
		)
	}
}

func TestOfCountsTheWordsOfTheTextAndOfTheTitleInEveryPosting(t *testing.T) {
	index := indexOf(t, sampleText)

	if len(index.Postings) == 0 {
		t.Fatal("no postings")
	}
	for _, posting := range index.Postings {
		if posting.TextWords != 6 {
			t.Fatalf("text words = %d, want the 6 words of %q", posting.TextWords, sampleText)
		}
		if posting.TitleWords != 2 {
			t.Fatalf("title words = %d, want the 2 words of %q", posting.TitleWords, documentTitle)
		}
	}
}

func TestOfCountsTitleWordsBetweenSpacesAsYaCyDoes(t *testing.T) {
	document := extractedDocument()
	document.Title = "Hello · World"

	index := pagerwi.Of(scrapedPage(t, sampleText), document, []byte(sampleText), reachedAt)

	if got := index.Postings[0].TitleWords; got != 3 {
		t.Fatalf("title words = %d, want the 3 space-separated words of %q", got, document.Title)
	}
}

func TestOfMarksAWordOfTheTitleAsAppearingInIt(t *testing.T) {
	index := indexOf(t, "hello there quick fox")

	if !postingOf(t, index, "hello").Appearance.AppearsInTitle {
		t.Errorf("word %q of title %q should appear in the title", "hello", documentTitle)
	}
	if postingOf(t, index, "quick").Appearance.AppearsInTitle {
		t.Errorf(
			"word %q is absent from title %q and should not appear in it",
			"quick",
			documentTitle,
		)
	}
}

func postingOf(t *testing.T, index pagerwi.PageRWI, word string) yacymodel.RWIPosting {
	t.Helper()

	wordHash := yacymodel.WordHash(word)
	for _, posting := range index.Postings {
		if posting.WordHash == wordHash {
			return posting
		}
	}
	t.Fatalf("word %q should be indexed", word)

	return yacymodel.RWIPosting{}
}

func TestOfMarksNoWordAsAppearingInAnAbsentTitle(t *testing.T) {
	document := extractedDocument()
	document.Title = ""

	index := pagerwi.Of(scrapedPage(t, sampleText), document, []byte(sampleText), reachedAt)

	for _, posting := range index.Postings {
		if posting.Appearance.AppearsInTitle {
			t.Fatalf("word %q should not appear in an absent title", posting.WordHash)
		}
	}
}

func assertWordIndexed(t *testing.T, text string, word string) {
	t.Helper()

	wordHash := yacymodel.WordHash(word)
	for _, posting := range indexOf(t, text).Postings {
		if posting.WordHash == wordHash {
			return
		}
	}
	t.Fatalf("word %q should be indexed", word)
}

func indexOfPageModifiedAt(t *testing.T, modifiedAt time.Time) pagerwi.PageRWI {
	t.Helper()
	scraped := scrapedPage(t, sampleText)
	scraped.PageModifiedAt = modifiedAt

	return pagerwi.Of(scraped, extractedDocument(), []byte(sampleText), reachedAt)
}

func microDateOf(t *testing.T, instant time.Time) yacymodel.MicroDate {
	t.Helper()

	date, ok := yacymodel.MicroDateOf(instant).Get()
	if !ok {
		t.Fatalf("MicroDateOf(%v) is absent", instant)
	}

	return date
}

func TestOfRecordsTheDateTheOriginSaidThePageWasModified(t *testing.T) {
	modifiedAt := reachedAt.Add(-72 * time.Hour)

	index := indexOfPageModifiedAt(t, modifiedAt)

	if modified, ok := index.Metadata.Modified.Get(); !ok ||
		modified != yacymodel.CalendarDayOf(modifiedAt) {
		t.Fatalf("metadata modified = %+v, want %v", index.Metadata.Modified, modifiedAt)
	}
	for _, posting := range index.Postings {
		if posting.LastModified != microDateOf(t, modifiedAt) {
			t.Fatalf("posting last modified = %v, want %v",
				posting.LastModified, microDateOf(t, modifiedAt))
		}
	}
}

func TestOfKeepsTheTimeThePageWasFetchedAsTheLoadedDate(t *testing.T) {
	index := indexOfPageModifiedAt(t, reachedAt.Add(-72*time.Hour))

	if loaded, ok := index.Metadata.Loaded.Get(); !ok ||
		loaded != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("metadata loaded = %+v, want %v", index.Metadata.Loaded, reachedAt)
	}
}

func TestOfFallsBackToTheTimeThePageWasReachedWhenTheOriginSaidNothing(t *testing.T) {
	index := indexOfPageModifiedAt(t, time.Time{})

	if modified, ok := index.Metadata.Modified.Get(); !ok ||
		modified != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("metadata modified = %+v, want %v", index.Metadata.Modified, reachedAt)
	}
	for _, posting := range index.Postings {
		if posting.LastModified != microDateOf(t, reachedAt) {
			t.Fatalf("posting last modified = %v, want %v",
				posting.LastModified, microDateOf(t, reachedAt))
		}
	}
}

func TestOfHoldsBackAModifiedDateLaterThanTheTimeThePageWasReached(t *testing.T) {
	index := indexOfPageModifiedAt(t, reachedAt.Add(240*time.Hour))

	if modified, ok := index.Metadata.Modified.Get(); !ok ||
		modified != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("metadata modified = %+v, want %v", index.Metadata.Modified, reachedAt)
	}
	for _, posting := range index.Postings {
		if posting.LastModified > microDateOf(t, reachedAt) {
			t.Fatalf("posting last modified = %v, want no later than %v",
				posting.LastModified, microDateOf(t, reachedAt))
		}
	}
}

func TestOfFallsBackToTheTimeThePageWasReachedForAModifiedDateBeforeTheEpoch(t *testing.T) {
	index := indexOfPageModifiedAt(t, time.Date(1965, time.March, 4, 0, 0, 0, 0, time.UTC))

	if modified, ok := index.Metadata.Modified.Get(); !ok ||
		modified != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("metadata modified = %+v, want %v", index.Metadata.Modified, reachedAt)
	}
	if got := index.Postings[0].LastModified; got != microDateOf(t, reachedAt) {
		t.Fatalf("posting last modified = %v, want %v", got, microDateOf(t, reachedAt))
	}
}

const amountOfOversizedPageSentences = 65600

func indexOfOversizedPage(t *testing.T) pagerwi.PageRWI {
	t.Helper()
	text := strings.Repeat("fox. ", amountOfOversizedPageSentences) + "zebra."
	document := extractedDocument()
	document.LocalLinks = 300
	document.ExternalLinks = 256

	return pagerwi.Of(scrapedPage(t, text), document, []byte(text), reachedAt)
}

func TestOfKeepsTheLowByteOfOneByteCountsOfAnOversizedPage(t *testing.T) {
	index := indexOfOversizedPage(t)
	fox, zebra := postingOf(t, index, "fox"), postingOf(t, index, "zebra")

	if fox.Hits != 64 {
		t.Errorf("hits = %d, want the low byte 64 of %d", fox.Hits, amountOfOversizedPageSentences)
	}
	if zebra.PhrasePosition != 164 {
		t.Errorf("phrase position = %d, want the low byte 164", zebra.PhrasePosition)
	}
	if fox.LocalLinks != 44 || fox.ExternalLinks != 0 {
		t.Errorf("links = %d local, %d external, want the low bytes 44 and 0",
			fox.LocalLinks, fox.ExternalLinks)
	}
	if index.Metadata.LocalLinks != 300 {
		t.Errorf(
			"metadata local links = %d, want the 300 the page holds",
			index.Metadata.LocalLinks,
		)
	}
}

func TestOfHoldsTwoByteCountsOfAnOversizedPageAtTheirCeiling(t *testing.T) {
	zebra := postingOf(t, indexOfOversizedPage(t), "zebra")

	if zebra.TextWords != math.MaxUint16 || zebra.Phrases != math.MaxUint16 {
		t.Errorf("text words = %d, phrases = %d, want both at %d",
			zebra.TextWords, zebra.Phrases, math.MaxUint16)
	}
	if zebra.TextPosition != math.MaxUint16 {
		t.Errorf("text position = %d, want %d", zebra.TextPosition, math.MaxUint16)
	}
}
