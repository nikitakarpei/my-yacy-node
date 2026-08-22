package pagerwi_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
)

const (
	sampleText             = "the quick brown fox the fox"
	documentMarkupByteSize = 512
)

var reachedAt = time.Unix(1_700_000_000, 0)

func samplePage(t *testing.T, text string) pagescrape.ScrapedPage {
	t.Helper()
	return pagescrape.ScrapedPage{
		CanonicalURL:     canonicalurltest.CanonicalURLOf(t, "http://example.com/article"),
		Title:            "Hello World",
		Language:         "en",
		LocalLinks:       3,
		ExternalLinks:    1,
		DocumentByteSize: len(text) + documentMarkupByteSize,
		Content:          []byte(text),
	}
}

func indexOf(t *testing.T, page pagescrape.ScrapedPage) pagerwi.PageRWI {
	t.Helper()

	return pagerwi.Of(page, reachedAt)
}

func TestOfProducesPostingsCarryingTheURLHash(t *testing.T) {
	index := indexOf(t, samplePage(t, sampleText))

	if index.CanonicalURL.String() != "http://example.com/article" {
		t.Fatalf("canonical url: %q", index.CanonicalURL)
	}
	if len(index.Postings) == 0 {
		t.Fatal("no postings")
	}
	urlHash := hashOfCanonicalURL(t)
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
	index := indexOf(t, samplePage(t, sampleText))

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
	page := samplePage(t, sampleText)
	metadata := indexOf(t, page).Metadata

	if metadata.Address != page.CanonicalURL.String() {
		t.Fatalf("address = %q", metadata.Address)
	}
	if metadata.Title != page.Title {
		t.Fatalf("title = %q", metadata.Title)
	}
	if metadata.ByteSize != page.DocumentByteSize {
		t.Fatalf("byte size = %d, want %d", metadata.ByteSize, page.DocumentByteSize)
	}
	if metadata.WordCount == 0 {
		t.Fatal("word count = 0, want nonzero")
	}
	if metadata.LocalLinks != page.LocalLinks ||
		metadata.ExternalLinks != page.ExternalLinks {
		t.Fatalf("links = %d local, %d external", metadata.LocalLinks, metadata.ExternalLinks)
	}
	if loaded, ok := metadata.Loaded.Get(); !ok || loaded != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("loaded = %+v", metadata.Loaded)
	}
}

func TestOfMetadataCarriesURLHash(t *testing.T) {
	got, err := indexOf(t, samplePage(t, sampleText)).Metadata.Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if want := hashOfCanonicalURL(t); got != want {
		t.Fatalf("metadata url hash = %q, want %q", got, want)
	}
}

func TestOfOmitsLanguageWhenAbsent(t *testing.T) {
	page := samplePage(t, sampleText)
	page.Language = ""

	for _, posting := range indexOf(t, page).Postings {
		if language, ok := posting.Language.Get(); ok {
			t.Fatalf("language should be empty when unknown, got %q", language)
		}
	}
}

func TestOfDropsWordsShorterThanTwoCharacters(t *testing.T) {
	index := indexOf(t, samplePage(t, "a fox I saw"))

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
	assertWordIndexed(t, "navigation menu the quick fox", "navigation")
}

func TestOfMetadataByteSizeReflectsTheFetchedDocument(t *testing.T) {
	page := samplePage(t, "the quick fox")

	index := indexOf(t, page)

	if index.Metadata.ByteSize != page.DocumentByteSize {
		t.Fatalf(
			"byte size = %d, want the fetched document size %d",
			index.Metadata.ByteSize,
			page.DocumentByteSize,
		)
	}
}

func TestOfCountsPhrasesAndPhrasePositions(t *testing.T) {
	index := indexOf(t, samplePage(t, "the quick fox jumps. the lazy dog sleeps."))

	for _, posting := range index.Postings {
		if posting.Phrases != 2 {
			t.Fatalf("phrase count = %d, want 2", posting.Phrases)
		}
	}
	jumpsHash := yacymodel.WordHash("jumps")
	sleepsHash := yacymodel.WordHash("sleeps")
	var jumpsPhrase, sleepsPhrase int
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

func assertWordIndexed(t *testing.T, text string, word string) {
	t.Helper()

	wordHash := yacymodel.WordHash(word)
	for _, posting := range indexOf(t, samplePage(t, text)).Postings {
		if posting.WordHash == wordHash {
			return
		}
	}
	t.Fatalf("word %q should be indexed", word)
}

func hashOfCanonicalURL(t *testing.T) yacymodel.URLHash {
	t.Helper()

	address, err := url.Parse(samplePage(t, sampleText).CanonicalURL.String())
	if err != nil {
		t.Fatalf("parse canonical url: %v", err)
	}

	return yacymodel.URLNormalformOf(address).Hash()
}
