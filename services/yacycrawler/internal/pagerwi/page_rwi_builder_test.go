package pagerwi_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type textDerivation struct{}

func (textDerivation) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatText
}

func (textDerivation) SourceFormats() []crawlcapability.PageContentFormat {
	return []crawlcapability.PageContentFormat{crawlcapability.PageContentFormatText}
}

func (textDerivation) Derive(
	body []byte,
	_ crawlcapability.PageContentFormat,
) ([]byte, error) {
	return body, nil
}

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

func TestBuildProducesParseablePostings(t *testing.T) {
	index, err := pagerwi.Build(samplePage(), textDerivation{})
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
	index, err := pagerwi.Build(samplePage(), textDerivation{})
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

func TestBuildMetadataParseableAndCarriesURLHash(t *testing.T) {
	index, err := pagerwi.Build(samplePage(), textDerivation{})
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
	urlHash, err := yacymodel.HashURL("http://example.com/article")
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
	index, err := pagerwi.Build(page, textDerivation{})
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
	index, err := pagerwi.Build(page, textDerivation{})
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
	page.Body = []byte("a fox I saw")
	index, err := pagerwi.Build(page, textDerivation{})
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
	page.Body = []byte("state-of-the-art design")
	index, err := pagerwi.Build(page, textDerivation{})
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
	page.Body = []byte("the price is 1,234.56 today")
	index, err := pagerwi.Build(page, textDerivation{})
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
	page.Body = []byte("the quick fox jumps. the lazy dog sleeps.")
	index, err := pagerwi.Build(page, textDerivation{})
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

type failingDerivation struct{}

func (failingDerivation) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatText
}

func (failingDerivation) SourceFormats() []crawlcapability.PageContentFormat {
	return []crawlcapability.PageContentFormat{crawlcapability.PageContentFormatText}
}

func (failingDerivation) Derive(
	[]byte,
	crawlcapability.PageContentFormat,
) ([]byte, error) {
	return nil, errors.New("malformed body")
}

func TestBuildFailsWhenTextDerivationFails(t *testing.T) {
	if _, err := pagerwi.Build(samplePage(), failingDerivation{}); err == nil {
		t.Fatal("expected error when the text derivation fails")
	}
}
