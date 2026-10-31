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

	ringPartitionExponent = 4
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

type noStopwords struct{}

func (noStopwords) Contains(yacymodel.Hash, yacymodel.Language) bool { return false }

func indexBuilderWithoutStopwords(t *testing.T) pagerwi.IndexBuilder {
	t.Helper()

	return indexBuilderOf(t, noStopwords{})
}

func indexBuilderOf(t *testing.T, stopwords pagerwi.Stopwords) pagerwi.IndexBuilder {
	t.Helper()

	return pagerwi.NewIndexBuilder(stopwords, ringPartitions(t))
}

func ringPartitions(t *testing.T) yacymodel.DHTRingPartitions {
	t.Helper()
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(ringPartitionExponent)
	if err != nil {
		t.Fatal(err)
	}

	return partitions
}

func withPagePostings(t *testing.T, pageRWI pagerwi.PageRWI) pagerwi.PageRWI {
	t.Helper()
	if !pageRWI.PagePostings().Present() {
		t.Fatal("the page has no page postings")
	}

	return pageRWI
}

func postingsOf(pageRWI pagerwi.PageRWI) []yacymodel.RWIPosting {
	pagePostings, _ := pageRWI.PagePostings().Get()

	return pagePostings.Postings()
}

func pageRWIOf(t *testing.T, text string) pagerwi.PageRWI {
	t.Helper()
	return withPagePostings(
		t,
		indexBuilderWithoutStopwords(
			t,
		).RWIFrom(t.Context(), scrapedPage(t, text), extractedDocument(), []byte(text), reachedAt),
	)
}

func TestRWIFromIndexesUnderThePageURLRatherThanTheURLTheFetchLandedOn(t *testing.T) {
	const replayURL = "http://archive.example/replay/http://example.com/"
	scraped := scrapedPage(t, sampleText)
	scraped.LandedURL = canonicalurltest.CanonicalURLOf(t, replayURL)

	pageRWI := withPagePostings(
		t,
		indexBuilderWithoutStopwords(
			t,
		).RWIFrom(t.Context(), scraped, extractedDocument(), []byte(sampleText), reachedAt),
	)

	if pageRWI.Metadata().Address != pageURL {
		t.Fatalf("metadata address = %q, want %q", pageRWI.Metadata().Address, pageURL)
	}
}

func TestRWIFromProducesPostingsCarryingTheURLHash(t *testing.T) {
	pageRWI := pageRWIOf(t, sampleText)

	if len(postingsOf(pageRWI)) == 0 {
		t.Fatal("no postings")
	}
	urlHash := urlhashtest.URLHashOf(t, pageURL)
	for _, posting := range postingsOf(pageRWI) {
		if posting.URLHash != urlHash {
			t.Fatalf("posting url hash = %q, want %q", posting.URLHash, urlHash)
		}
		if posting.DocumentType != yacymodel.DocumentTypeText {
			t.Fatalf("posting document type = %v, want text", posting.DocumentType)
		}
	}
}

func TestRWIFromCountsRepeatedWords(t *testing.T) {
	pageRWI := pageRWIOf(t, sampleText)

	foxHash := yacymodel.WordHash("fox")
	var found bool
	for _, posting := range postingsOf(pageRWI) {
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

func TestRWIFromCarriesTextStatsAndPageReferenceIntoMetadata(t *testing.T) {
	document := extractedDocument()
	metadata := pageRWIOf(t, sampleText).Metadata()

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

func TestRWIFromRecordsTheUndeclaredLanguageInEveryPosting(t *testing.T) {
	document := extractedDocument()
	document.Language = ""

	pageRWI := withPagePostings(
		t,
		indexBuilderWithoutStopwords(
			t,
		).RWIFrom(t.Context(), scrapedPage(t, sampleText), document, []byte(sampleText), reachedAt),
	)

	for _, posting := range postingsOf(pageRWI) {
		if posting.Language != yacymodel.LanguageOfUndeclaredDocument {
			t.Fatalf("language = %q, want %q",
				posting.Language, yacymodel.LanguageOfUndeclaredDocument)
		}
	}
}

func TestRWIFromLeavesMetadataLanguageAbsentWhenThePageDeclaresNone(t *testing.T) {
	document := extractedDocument()
	document.Language = ""

	pageRWI := withPagePostings(
		t,
		indexBuilderWithoutStopwords(
			t,
		).RWIFrom(t.Context(), scrapedPage(t, sampleText), document, []byte(sampleText), reachedAt),
	)

	if language, ok := pageRWI.Metadata().Language.Get(); ok {
		t.Fatalf("metadata language = %q, want absent", language)
	}
}

func TestRWIFromDropsWordsShorterThanTwoCharacters(t *testing.T) {
	pageRWI := pageRWIOf(t, "a fox I saw")

	for _, posting := range postingsOf(pageRWI) {
		if posting.WordHash == yacymodel.WordHash("a") ||
			posting.WordHash == yacymodel.WordHash("i") {
			t.Fatalf("short word should not be indexed: %q", posting.WordHash)
		}
	}
	if len(postingsOf(pageRWI)) != 2 {
		t.Fatalf("want postings for 'fox' and 'saw' only, got %d", len(postingsOf(pageRWI)))
	}
}

func TestRWIFromKeepsHyphenatedCompoundAsOneWord(t *testing.T) {
	assertWordIndexed(t, "state-of-the-art design", "state-of-the-art")
}

func TestRWIFromKeepsDigitSeparatedNumberAsOneWord(t *testing.T) {
	assertWordIndexed(t, "the price is 1,234.56 today", "1,234.56")
}

func TestRWIFromIndexesEveryWordOfThePageText(t *testing.T) {
	words := []string{"navigation", "menu", "the", "quick", "fox"}

	pageRWI := pageRWIOf(t, strings.Join(words, " "))

	if len(postingsOf(pageRWI)) != len(words) {
		t.Fatalf("postings = %d, want one per word of %v", len(postingsOf(pageRWI)), words)
	}
	for _, word := range words {
		assertWordIndexed(t, strings.Join(words, " "), word)
	}
}

func TestRWIFromMetadataByteSizeReflectsTheFetchedDocument(t *testing.T) {
	page := scrapedPage(t, "the quick fox")

	pageRWI := withPagePostings(
		t,
		indexBuilderWithoutStopwords(
			t,
		).RWIFrom(t.Context(), page, extractedDocument(), []byte("the quick fox"), reachedAt),
	)

	if size, known := pageRWI.Metadata().ByteSize.Get(); !known || int(size) != len(page.Body) {
		t.Fatalf(
			"byte size = %v, want the fetched document size %d",
			pageRWI.Metadata().ByteSize,
			len(page.Body),
		)
	}
}

func TestRWIFromPostingsMeasureThePageURL(t *testing.T) {
	pageRWI := pageRWIOf(t, sampleText)

	for _, posting := range postingsOf(pageRWI) {
		if int(posting.URLLength) != len(pageURL) {
			t.Fatalf("url length = %d, want %d", posting.URLLength, len(pageURL))
		}
		if posting.URLComponents != 3 {
			t.Fatalf("url components = %d, want 3", posting.URLComponents)
		}
	}
}

func TestRWIFromCountsPhrasesAndPhrasePositions(t *testing.T) {
	pageRWI := pageRWIOf(t, "the quick fox jumps. the lazy dog sleeps.")

	for _, posting := range postingsOf(pageRWI) {
		if posting.Phrases != 2 {
			t.Fatalf("phrase count = %d, want 2", posting.Phrases)
		}
	}
	jumpsHash := yacymodel.WordHash("jumps")
	sleepsHash := yacymodel.WordHash("sleeps")
	var jumpsPhrase, sleepsPhrase uint8
	var jumpsFound, sleepsFound bool
	for _, posting := range postingsOf(pageRWI) {
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

func TestRWIFromCountsTheWordsOfTheTextAndOfTheTitleInEveryPosting(t *testing.T) {
	pageRWI := pageRWIOf(t, sampleText)

	if len(postingsOf(pageRWI)) == 0 {
		t.Fatal("no postings")
	}
	for _, posting := range postingsOf(pageRWI) {
		if posting.TextWords != 6 {
			t.Fatalf("text words = %d, want the 6 words of %q", posting.TextWords, sampleText)
		}
		if posting.TitleWords != 2 {
			t.Fatalf("title words = %d, want the 2 words of %q", posting.TitleWords, documentTitle)
		}
	}
}

func TestRWIFromCountsTitleWordsBetweenSpacesAsYaCyDoes(t *testing.T) {
	document := extractedDocument()
	document.Title = "Hello · World"

	pageRWI := withPagePostings(
		t,
		indexBuilderWithoutStopwords(
			t,
		).RWIFrom(t.Context(), scrapedPage(t, sampleText), document, []byte(sampleText), reachedAt),
	)

	if got := postingsOf(pageRWI)[0].TitleWords; got != 3 {
		t.Fatalf("title words = %d, want the 3 space-separated words of %q", got, document.Title)
	}
}

func TestRWIFromMarksAWordOfTheTitleAsAppearingInIt(t *testing.T) {
	pageRWI := pageRWIOf(t, "hello there quick fox")

	if !postingOf(t, pageRWI, "hello").Appearance.AppearsInTitle {
		t.Errorf("word %q of title %q should appear in the title", "hello", documentTitle)
	}
	if postingOf(t, pageRWI, "quick").Appearance.AppearsInTitle {
		t.Errorf(
			"word %q is absent from title %q and should not appear in it",
			"quick",
			documentTitle,
		)
	}
}

func postingOf(t *testing.T, pageRWI pagerwi.PageRWI, word string) yacymodel.RWIPosting {
	t.Helper()

	wordHash := yacymodel.WordHash(word)
	for _, posting := range postingsOf(pageRWI) {
		if posting.WordHash == wordHash {
			return posting
		}
	}
	t.Fatalf("word %q should be indexed", word)

	return yacymodel.RWIPosting{}
}

func TestRWIFromMarksNoWordAsAppearingInAnAbsentTitle(t *testing.T) {
	document := extractedDocument()
	document.Title = ""

	pageRWI := withPagePostings(
		t,
		indexBuilderWithoutStopwords(
			t,
		).RWIFrom(t.Context(), scrapedPage(t, sampleText), document, []byte(sampleText), reachedAt),
	)

	for _, posting := range postingsOf(pageRWI) {
		if posting.Appearance.AppearsInTitle {
			t.Fatalf("word %q should not appear in an absent title", posting.WordHash)
		}
	}
}

func assertWordIndexed(t *testing.T, text string, word string) {
	t.Helper()

	wordHash := yacymodel.WordHash(word)
	for _, posting := range postingsOf(pageRWIOf(t, text)) {
		if posting.WordHash == wordHash {
			return
		}
	}
	t.Fatalf("word %q should be indexed", word)
}

func pageRWIOfPageModifiedAt(t *testing.T, modifiedAt time.Time) pagerwi.PageRWI {
	t.Helper()
	scraped := scrapedPage(t, sampleText)
	scraped.PageModifiedAt = modifiedAt

	return withPagePostings(
		t,
		indexBuilderWithoutStopwords(
			t,
		).RWIFrom(t.Context(), scraped, extractedDocument(), []byte(sampleText), reachedAt),
	)
}

func microDateOf(t *testing.T, instant time.Time) yacymodel.MicroDate {
	t.Helper()

	date, ok := yacymodel.MicroDateOf(instant).Get()
	if !ok {
		t.Fatalf("MicroDateOf(%v) is absent", instant)
	}

	return date
}

func TestRWIFromRecordsTheDateTheOriginSaidThePageWasModified(t *testing.T) {
	modifiedAt := reachedAt.Add(-72 * time.Hour)

	pageRWI := pageRWIOfPageModifiedAt(t, modifiedAt)

	if modified, ok := pageRWI.Metadata().Modified.Get(); !ok ||
		modified != yacymodel.CalendarDayOf(modifiedAt) {
		t.Fatalf("metadata modified = %+v, want %v", pageRWI.Metadata().Modified, modifiedAt)
	}
	for _, posting := range postingsOf(pageRWI) {
		if posting.LastModified != microDateOf(t, modifiedAt) {
			t.Fatalf("posting last modified = %v, want %v",
				posting.LastModified, microDateOf(t, modifiedAt))
		}
	}
}

func TestRWIFromKeepsTheTimeThePageWasFetchedAsTheLoadedDate(t *testing.T) {
	pageRWI := pageRWIOfPageModifiedAt(t, reachedAt.Add(-72*time.Hour))

	if loaded, ok := pageRWI.Metadata().Loaded.Get(); !ok ||
		loaded != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("metadata loaded = %+v, want %v", pageRWI.Metadata().Loaded, reachedAt)
	}
}

func TestRWIFromFallsBackToTheTimeThePageWasReachedWhenTheOriginSaidNothing(t *testing.T) {
	pageRWI := pageRWIOfPageModifiedAt(t, time.Time{})

	if modified, ok := pageRWI.Metadata().Modified.Get(); !ok ||
		modified != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("metadata modified = %+v, want %v", pageRWI.Metadata().Modified, reachedAt)
	}
	for _, posting := range postingsOf(pageRWI) {
		if posting.LastModified != microDateOf(t, reachedAt) {
			t.Fatalf("posting last modified = %v, want %v",
				posting.LastModified, microDateOf(t, reachedAt))
		}
	}
}

func TestRWIFromHoldsBackAModifiedDateLaterThanTheTimeThePageWasReached(t *testing.T) {
	pageRWI := pageRWIOfPageModifiedAt(t, reachedAt.Add(240*time.Hour))

	if modified, ok := pageRWI.Metadata().Modified.Get(); !ok ||
		modified != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("metadata modified = %+v, want %v", pageRWI.Metadata().Modified, reachedAt)
	}
	for _, posting := range postingsOf(pageRWI) {
		if posting.LastModified > microDateOf(t, reachedAt) {
			t.Fatalf("posting last modified = %v, want no later than %v",
				posting.LastModified, microDateOf(t, reachedAt))
		}
	}
}

func TestRWIFromFallsBackToTheTimeThePageWasReachedForAModifiedDateBeforeTheEpoch(t *testing.T) {
	pageRWI := pageRWIOfPageModifiedAt(t, time.Date(1965, time.March, 4, 0, 0, 0, 0, time.UTC))

	if modified, ok := pageRWI.Metadata().Modified.Get(); !ok ||
		modified != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("metadata modified = %+v, want %v", pageRWI.Metadata().Modified, reachedAt)
	}
	if got := postingsOf(pageRWI)[0].LastModified; got != microDateOf(t, reachedAt) {
		t.Fatalf("posting last modified = %v, want %v", got, microDateOf(t, reachedAt))
	}
}

const amountOfOversizedPageSentences = 65600

func pageRWIOfOversizedPage(t *testing.T) pagerwi.PageRWI {
	t.Helper()
	text := strings.Repeat("fox. ", amountOfOversizedPageSentences) + "zebra."
	document := extractedDocument()
	document.LocalLinks = 300
	document.ExternalLinks = 256

	return withPagePostings(
		t,
		indexBuilderWithoutStopwords(
			t,
		).RWIFrom(t.Context(), scrapedPage(t, text), document, []byte(text), reachedAt),
	)
}

func TestRWIFromKeepsTheLowByteOfOneByteCountsOfAnOversizedPage(t *testing.T) {
	pageRWI := pageRWIOfOversizedPage(t)
	fox, zebra := postingOf(t, pageRWI, "fox"), postingOf(t, pageRWI, "zebra")

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
	if pageRWI.Metadata().LocalLinks != 300 {
		t.Errorf(
			"metadata local links = %d, want the 300 the page holds",
			pageRWI.Metadata().LocalLinks,
		)
	}
}

func TestRWIFromHoldsTwoByteCountsOfAnOversizedPageAtTheirCeiling(t *testing.T) {
	zebra := postingOf(t, pageRWIOfOversizedPage(t), "zebra")

	if zebra.TextWords != math.MaxUint16 || zebra.Phrases != math.MaxUint16 {
		t.Errorf("text words = %d, phrases = %d, want both at %d",
			zebra.TextWords, zebra.Phrases, math.MaxUint16)
	}
	if zebra.TextPosition != math.MaxUint16 {
		t.Errorf("text position = %d, want %d", zebra.TextPosition, math.MaxUint16)
	}
}

type stopwordThe struct{}

func (stopwordThe) Contains(word yacymodel.Hash, _ yacymodel.Language) bool {
	return word == yacymodel.WordHash("the")
}

func TestRWIFromBuildsNoPostingOfAStopword(t *testing.T) {
	pageRWI := withPagePostings(t, indexBuilderOf(t, stopwordThe{}).RWIFrom(t.Context(),
		scrapedPage(t, sampleText),
		extractedDocument(),
		[]byte(sampleText),
		reachedAt,
	))

	for _, posting := range postingsOf(pageRWI) {
		if posting.WordHash == yacymodel.WordHash("the") {
			t.Fatal("the stopword of the page has a posting, want none")
		}
	}
	if len(postingsOf(pageRWI)) != 3 {
		t.Fatalf("postings = %d, want quick, brown and fox", len(postingsOf(pageRWI)))
	}
	if textWords := postingsOf(pageRWI)[0].TextWords; textWords != 6 {
		t.Fatalf(
			"text words = %d, want the 6 words of %q with its stopwords",
			textWords,
			sampleText,
		)
	}
}

func TestRWIFromRootsTheMetadataAtThePostingsWithoutStopwords(t *testing.T) {
	pageRWI := withPagePostings(t, indexBuilderOf(t, stopwordThe{}).RWIFrom(t.Context(),
		scrapedPage(t, sampleText),
		extractedDocument(),
		[]byte(sampleText),
		reachedAt,
	))

	kept, err := yacymodel.ProvenPagePostingsFrom(
		postingsOf(pageRWI),
		yacymodel.EmptyPostingsProof(ringPartitions(t)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := pageRWI.Metadata().PostingsRoot; !got.Present() || got != kept.Root() {
		t.Fatalf(
			"metadata postings root = %+v, want the root %+v of the kept postings",
			got,
			kept.Root(),
		)
	}
}

func TestRWIFromAPageOfStopwordsOnlyHasMetadataWithoutPagePostingsOrRoot(t *testing.T) {
	pageRWI := indexBuilderOf(t, stopwordThe{}).RWIFrom(t.Context(),
		scrapedPage(t, "the the"),
		extractedDocument(),
		[]byte("the the"),
		reachedAt,
	)

	if pageRWI.PagePostings().Present() {
		t.Errorf("page postings = %+v, want none for a page of stopwords only",
			pageRWI.PagePostings())
	}
	metadata := pageRWI.Metadata()
	if metadata.Hash != urlhashtest.URLHashOf(t, pageURL) || metadata.PostingsRoot.Present() {
		t.Errorf("metadata = %+v, want the page url hash and no postings root", metadata)
	}
}
