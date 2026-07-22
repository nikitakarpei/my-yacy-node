package rwirepresentation_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/rwirepresentation"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func samplePage() crawlcapability.CrawledPage {
	return crawlcapability.CrawledPage{
		CanonicalURL:      "http://example.com/article",
		Title:             "Hello World",
		Body:              []byte("the quick brown fox the fox"),
		Format:            crawlcapability.PageContentFormatDocumentHTML,
		Language:          "en",
		CrawledAt:         time.Unix(1_700_000_000, 0),
		LocalLinkCount:    3,
		ExternalLinkCount: 1,
	}
}

const sampleText = "the quick brown fox the fox"

func TestBuildProducesParseablePostings(t *testing.T) {
	index, err := rwirepresentation.Build(samplePage(), []byte(sampleText))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if index.CanonicalURL != "http://example.com/article" {
		t.Fatalf("canonical url: %q", index.CanonicalURL)
	}
	if len(index.Postings) == 0 {
		t.Fatal("no postings")
	}
	urlHash, err := yacymodel.HashURL(samplePage().CanonicalURL)
	if err != nil {
		t.Fatal(err)
	}
	for _, posting := range index.Postings {
		if posting.URLHash != urlHash {
			t.Fatalf("posting url hash = %q, want %q", posting.URLHash, urlHash)
		}
		if posting.DocumentType != yacymodel.DocumentTypeText {
			t.Fatalf("posting document type = %v, want text", posting.DocumentType)
		}
	}
}

func TestBuildCountsRepeatedWords(t *testing.T) {
	index, err := rwirepresentation.Build(samplePage(), []byte(sampleText))
	if err != nil {
		t.Fatal(err)
	}
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

func TestBuildCarriesTextStatsAndPageReferenceIntoMetadata(t *testing.T) {
	page := samplePage()
	index, err := rwirepresentation.Build(page, []byte(sampleText))
	if err != nil {
		t.Fatal(err)
	}
	if index.CanonicalURL != page.CanonicalURL {
		t.Fatalf("canonical url = %q", index.CanonicalURL)
	}
	if len(index.Metadata) != 1 {
		t.Fatalf("want one metadata row, got %d", len(index.Metadata))
	}
	metadata := index.Metadata[0]
	if metadata.Address != page.CanonicalURL {
		t.Fatalf("address = %q", metadata.Address)
	}
	if metadata.Title != page.Title {
		t.Fatalf("title = %q", metadata.Title)
	}
	if metadata.ByteSize != len(sampleText) {
		t.Fatalf("byte size = %d, want %d", metadata.ByteSize, len(sampleText))
	}
	if metadata.WordCount == 0 {
		t.Fatal("word count = 0, want nonzero")
	}
	if metadata.LocalLinks != page.LocalLinkCount ||
		metadata.ExternalLinks != page.ExternalLinkCount {
		t.Fatalf("links = %d local, %d external", metadata.LocalLinks, metadata.ExternalLinks)
	}
	if loaded, ok := metadata.Loaded.Get(); !ok ||
		loaded != yacymodel.CalendarDayOf(page.CrawledAt) {
		t.Fatalf("loaded = %+v", metadata.Loaded)
	}
}

func TestBuildMetadataCarriesURLHash(t *testing.T) {
	index, err := rwirepresentation.Build(samplePage(), []byte(sampleText))
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Metadata) != 1 {
		t.Fatalf("want one metadata row, got %d", len(index.Metadata))
	}
	got, err := index.Metadata[0].Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	want, err := yacymodel.HashURL(samplePage().CanonicalURL)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("metadata url hash = %q, want %q", got, want)
	}
}

func TestBuildOmitsLanguageWhenAbsent(t *testing.T) {
	page := samplePage()
	page.Language = ""
	index, err := rwirepresentation.Build(page, []byte(sampleText))
	if err != nil {
		t.Fatal(err)
	}
	for _, posting := range index.Postings {
		if language, ok := posting.Language.Get(); ok {
			t.Fatalf("language should be empty when unknown, got %q", language)
		}
	}
}

func TestBuildDropsWordsShorterThanTwoCharacters(t *testing.T) {
	page := samplePage()
	index, err := rwirepresentation.Build(page, []byte("a fox I saw"))
	if err != nil {
		t.Fatal(err)
	}
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

func TestBuildKeepsHyphenatedCompoundAsOneWord(t *testing.T) {
	page := samplePage()
	index, err := rwirepresentation.Build(page, []byte("state-of-the-art design"))
	if err != nil {
		t.Fatal(err)
	}
	compoundHash := yacymodel.WordHash("state-of-the-art")
	var found bool
	for _, posting := range index.Postings {
		if posting.WordHash == compoundHash {
			found = true
		}
	}
	if !found {
		t.Fatal("hyphenated compound should be indexed as a single word")
	}
}

func TestBuildKeepsDigitSeparatedNumberAsOneWord(t *testing.T) {
	page := samplePage()
	index, err := rwirepresentation.Build(page, []byte("the price is 1,234.56 today"))
	if err != nil {
		t.Fatal(err)
	}
	numberHash := yacymodel.WordHash("1,234.56")
	var found bool
	for _, posting := range index.Postings {
		if posting.WordHash == numberHash {
			found = true
		}
	}
	if !found {
		t.Fatal("number with digit separators should be indexed as a single token")
	}
}

func TestBuildIndexesEveryWordOfGivenText(t *testing.T) {
	page := samplePage()
	fullText := []byte("navigation menu the quick fox")
	index, err := rwirepresentation.Build(page, fullText)
	if err != nil {
		t.Fatal(err)
	}
	byWord := map[yacymodel.Hash]bool{}
	for _, posting := range index.Postings {
		byWord[posting.WordHash] = true
	}
	if !byWord[yacymodel.WordHash("navigation")] {
		t.Fatal("every word of the given text should be indexed")
	}
}

func TestBuildMetadataByteSizeReflectsDocumentBody(t *testing.T) {
	page := samplePage()
	page.Body = []byte("<html><body>the quick fox</body></html>")
	index, err := rwirepresentation.Build(page, []byte("the quick fox"))
	if err != nil {
		t.Fatal(err)
	}
	if index.Metadata[0].ByteSize != len(page.Body) {
		t.Fatalf(
			"byte size = %d, want document body %d",
			index.Metadata[0].ByteSize,
			len(page.Body),
		)
	}
}

func TestBuildCountsPhrasesAndPhrasePositions(t *testing.T) {
	page := samplePage()
	index, err := rwirepresentation.Build(page, []byte("the quick fox jumps. the lazy dog sleeps."))
	if err != nil {
		t.Fatal(err)
	}
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
