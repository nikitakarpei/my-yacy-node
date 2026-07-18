package pagerwi_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func samplePage() crawlcapability.CrawledPage {
	return crawlcapability.CrawledPage{
		CanonicalURL:      "http://example.com/article",
		Title:             "Hello World",
		Body:              []byte("the quick brown fox the fox"),
		Format:            crawlcapability.PageContentFormatText,
		Language:          "en",
		CrawledAt:         time.Unix(1_700_000_000, 0),
		LocalLinkCount:    3,
		ExternalLinkCount: 1,
	}
}

const sampleText = "the quick brown fox the fox"

func TestBuildProducesParseablePostings(t *testing.T) {
	index, err := pagerwi.Build(samplePage(), []byte(sampleText))
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
	index, err := pagerwi.Build(samplePage(), []byte(sampleText))
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
	index, err := pagerwi.Build(page, []byte(sampleText))
	if err != nil {
		t.Fatal(err)
	}
	if index.CanonicalURL != page.CanonicalURL {
		t.Fatalf("canonical url = %q", index.CanonicalURL)
	}
	if len(index.Metadata) != 1 {
		t.Fatalf("want one metadata row, got %d", len(index.Metadata))
	}
	properties := index.Metadata[0].Properties
	if properties["url"] != yacymodel.EncodeBase64WireForm(page.CanonicalURL) {
		t.Fatalf("url = %q", properties["url"])
	}
	if properties[yacymodel.URLMetaColDescription] != yacymodel.EncodeBase64WireForm(page.Title) {
		t.Fatalf("title = %q", properties[yacymodel.URLMetaColDescription])
	}
	if properties["size"] != strconv.Itoa(len(sampleText)) {
		t.Fatalf("size = %q, want %d", properties["size"], len(sampleText))
	}
	if properties["wc"] == "" || properties["wc"] == "0" {
		t.Fatalf("word count = %q, want nonzero", properties["wc"])
	}
	if properties["llocal"] != strconv.Itoa(page.LocalLinkCount) ||
		properties["lother"] != strconv.Itoa(page.ExternalLinkCount) {
		t.Fatalf("link counts = %q %q", properties["llocal"], properties["lother"])
	}
}

func TestBuildMetadataParseableAndCarriesURLHash(t *testing.T) {
	index, err := pagerwi.Build(samplePage(), []byte(sampleText))
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Metadata) != 1 {
		t.Fatalf("want one metadata row, got %d", len(index.Metadata))
	}
	row := index.Metadata[0]
	if _, err := yacymodel.ParseURIMetadataRow(row.String()); err != nil {
		t.Fatalf("metadata row not parseable: %v", err)
	}
	urlHash, err := yacymodel.HashURL(samplePage().CanonicalURL)
	if err != nil {
		t.Fatal(err)
	}
	if row.Properties[yacymodel.URLMetaHash] != urlHash.String() {
		t.Fatalf("metadata url hash = %q, want %q",
			row.Properties[yacymodel.URLMetaHash], urlHash.String())
	}
}

func TestBuildMetadataSurvivesCommaInTitleAndURL(t *testing.T) {
	page := samplePage()
	page.CanonicalURL = "http://example.com/article?ids=1,2,3"
	page.Title = "Fourth of July fireworks, 1986 - Example"
	index, err := pagerwi.Build(page, []byte(sampleText))
	if err != nil {
		t.Fatal(err)
	}
	row := index.Metadata[0]
	parsed, err := yacymodel.ParseURIMetadataRow(row.String())
	if err != nil {
		t.Fatalf("metadata row not parseable: %v", err)
	}
	title, err := parsed.Title(t.Context())
	if err != nil {
		t.Fatalf("Title: %v", err)
	}
	if title != page.Title {
		t.Fatalf("title = %q, want %q", title, page.Title)
	}
}

func TestBuildOmitsLanguageWhenAbsent(t *testing.T) {
	page := samplePage()
	page.Language = ""
	index, err := pagerwi.Build(page, []byte(sampleText))
	if err != nil {
		t.Fatal(err)
	}
	for _, posting := range index.Postings {
		if posting.Language != "" {
			t.Fatalf("language should be empty when unknown, got %q", posting.Language)
		}
	}
}

func TestBuildDropsWordsShorterThanTwoCharacters(t *testing.T) {
	page := samplePage()
	index, err := pagerwi.Build(page, []byte("a fox I saw"))
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
	index, err := pagerwi.Build(page, []byte("state-of-the-art design"))
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
	index, err := pagerwi.Build(page, []byte("the price is 1,234.56 today"))
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

func TestBuildCountsPhrasesAndPhrasePositions(t *testing.T) {
	page := samplePage()
	index, err := pagerwi.Build(page, []byte("the quick fox jumps. the lazy dog sleeps."))
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
