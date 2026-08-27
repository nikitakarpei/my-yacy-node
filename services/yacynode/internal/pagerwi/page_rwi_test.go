package pagerwi_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
)

const (
	pageAddress   = "http://example.com/article"
	sampleText    = "the quick brown fox the fox"
	documentTitle = "Hello World"
)

var reachedAt = time.Unix(1_700_000_000, 0)

func fetchedPage(t *testing.T, text string) pagefetch.FetchedPage {
	t.Helper()
	return pagefetch.FetchedPage{
		LandedURL:   canonicalurltest.CanonicalURLOf(t, pageAddress),
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

func pageURLOf(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, pageAddress)
}

func indexOf(t *testing.T, text string) pagerwi.PageRWI {
	t.Helper()
	return pagerwi.Of(
		pageURLOf(t), fetchedPage(t, text), extractedDocument(), []byte(text), reachedAt,
	)
}

func TestOfIndexesUnderThePageURLRatherThanTheURLTheFetchLandedOn(t *testing.T) {
	const replayAddress = "http://archive.example/replay/http://example.com/"
	fetched := fetchedPage(t, sampleText)
	fetched.LandedURL = canonicalurltest.CanonicalURLOf(t, replayAddress)

	index := pagerwi.Of(
		pageURLOf(t), fetched, extractedDocument(), []byte(sampleText), reachedAt,
	)

	if index.PageURL.String() != pageAddress {
		t.Fatalf("page url = %q, want %q", index.PageURL, pageAddress)
	}
	if index.Metadata.Address != pageAddress {
		t.Fatalf("metadata address = %q, want %q", index.Metadata.Address, pageAddress)
	}
}

func TestOfProducesPostingsCarryingTheURLHash(t *testing.T) {
	index := indexOf(t, sampleText)

	if index.PageURL.String() != pageAddress {
		t.Fatalf("page url: %q", index.PageURL)
	}
	if len(index.Postings) == 0 {
		t.Fatal("no postings")
	}
	urlHash := hashOfPageAddress(t)
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

	if metadata.Address != pageAddress {
		t.Fatalf("address = %q", metadata.Address)
	}
	if metadata.Title != document.Title {
		t.Fatalf("title = %q", metadata.Title)
	}
	if metadata.WordCount == 0 {
		t.Fatal("word count = 0, want nonzero")
	}
	if metadata.LocalLinks != document.LocalLinks ||
		metadata.ExternalLinks != document.ExternalLinks {
		t.Fatalf("links = %d local, %d external", metadata.LocalLinks, metadata.ExternalLinks)
	}
	if loaded, ok := metadata.Loaded.Get(); !ok || loaded != yacymodel.CalendarDayOf(reachedAt) {
		t.Fatalf("loaded = %+v", metadata.Loaded)
	}
}

func TestOfMetadataCarriesURLHash(t *testing.T) {
	got, err := indexOf(t, sampleText).Metadata.Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if want := hashOfPageAddress(t); got != want {
		t.Fatalf("metadata url hash = %q, want %q", got, want)
	}
}

func TestOfOmitsLanguageWhenAbsent(t *testing.T) {
	document := extractedDocument()
	document.Language = ""

	index := pagerwi.Of(
		pageURLOf(t), fetchedPage(t, sampleText), document, []byte(sampleText), reachedAt,
	)

	for _, posting := range index.Postings {
		if language, ok := posting.Language.Get(); ok {
			t.Fatalf("language should be empty when unknown, got %q", language)
		}
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
	assertWordIndexed(t, "navigation menu the quick fox", "navigation")
}

func TestOfMetadataByteSizeReflectsTheFetchedDocument(t *testing.T) {
	page := fetchedPage(t, "the quick fox")

	index := pagerwi.Of(
		pageURLOf(t), page, extractedDocument(), []byte("the quick fox"), reachedAt,
	)

	if index.Metadata.ByteSize != len(page.Body) {
		t.Fatalf(
			"byte size = %d, want the fetched document size %d",
			index.Metadata.ByteSize,
			len(page.Body),
		)
	}
}

func TestOfPostingsMeasureThePageURL(t *testing.T) {
	index := indexOf(t, sampleText)

	for _, posting := range index.Postings {
		if posting.URLLength != len(pageAddress) {
			t.Fatalf("url length = %d, want %d", posting.URLLength, len(pageAddress))
		}
		if posting.URLComponents != 1 {
			t.Fatalf("url components = %d, want 1", posting.URLComponents)
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
	for _, posting := range indexOf(t, text).Postings {
		if posting.WordHash == wordHash {
			return
		}
	}
	t.Fatalf("word %q should be indexed", word)
}

func hashOfPageAddress(t *testing.T) yacymodel.URLHash {
	t.Helper()

	address, err := url.Parse(pageAddress)
	if err != nil {
		t.Fatalf("parse page address: %v", err)
	}

	return yacymodel.URLNormalformOf(address).Hash()
}
