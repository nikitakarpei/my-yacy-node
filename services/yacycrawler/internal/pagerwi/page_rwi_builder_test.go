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
	for _, posting := range index.Postings {
		if _, err := yacymodel.ParseRWIPosting(posting.String()); err != nil {
			t.Fatalf("posting %q not parseable: %v", posting.String(), err)
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
			if posting.Properties[yacymodel.ColHitCount] != "2" {
				t.Fatalf("fox hit count = %q, want 2", posting.Properties[yacymodel.ColHitCount])
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

func TestBuildOmitsLanguageWhenAbsent(t *testing.T) {
	page := samplePage()
	page.Language = ""
	index, err := pagerwi.Build(page, []byte(sampleText))
	if err != nil {
		t.Fatal(err)
	}
	for _, posting := range index.Postings {
		if _, ok := posting.Properties[yacymodel.ColLanguage]; ok {
			t.Fatal("language column should be omitted when unknown")
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
			t.Fatalf("short word should not be indexed: %q", posting.String())
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
		if posting.Properties[yacymodel.ColPhraseCount] != "2" {
			t.Fatalf("phrase count = %q, want 2", posting.Properties[yacymodel.ColPhraseCount])
		}
	}
	jumpsHash := yacymodel.WordHash("jumps")
	sleepsHash := yacymodel.WordHash("sleeps")
	var jumpsPhrase, sleepsPhrase string
	for _, posting := range index.Postings {
		if posting.WordHash == jumpsHash {
			jumpsPhrase = posting.Properties[yacymodel.ColPhrasePosition]
		}
		if posting.WordHash == sleepsHash {
			sleepsPhrase = posting.Properties[yacymodel.ColPhrasePosition]
		}
	}
	if jumpsPhrase == "" || jumpsPhrase == sleepsPhrase {
		t.Fatalf(
			"words in different sentences should get different phrase numbers, got %q and %q",
			jumpsPhrase,
			sleepsPhrase,
		)
	}
}
