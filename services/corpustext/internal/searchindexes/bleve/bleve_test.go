package bleve_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/bleve"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

func bootstrappedIndexes(t *testing.T, directory string) *bleve.Indexes {
	t.Helper()
	languageIndexes, err := languageindex.IndexesFor("yacy_text", []string{"en", "de"})
	if err != nil {
		t.Fatalf("language indexes: %v", err)
	}
	indexes := bleve.NewIndexes(directory, languageIndexes)
	if err := indexes.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	return indexes
}

func openIndexes(t *testing.T) *bleve.Indexes {
	t.Helper()
	indexes := bootstrappedIndexes(t, t.TempDir())
	t.Cleanup(func() { indexes.Close(context.Background()) })
	return indexes
}

func indexed(t *testing.T, indexes *bleve.Indexes, documents ...searchdocument.Document) {
	t.Helper()
	for _, document := range documents {
		if err := indexes.Index(context.Background(), document); err != nil {
			t.Fatalf("index %s: %v", document.URL, err)
		}
	}
}

func matchesFor(
	t *testing.T,
	indexes *bleve.Indexes,
	search textsearch.Search,
) []searchdocument.PageMatch {
	t.Helper()
	if search.Limit == 0 {
		search.Limit = textsearch.DefaultMatchLimit
	}
	matches, err := indexes.MatchesFor(context.Background(), search)
	if err != nil {
		t.Fatalf("matches: %v", err)
	}
	return matches
}

func urlsOf(matches []searchdocument.PageMatch) []string {
	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		urls = append(urls, match.URL)
	}
	return urls
}

func englishPage(url, title, content string) searchdocument.Document {
	return searchdocument.Document{
		URL: url, Title: title, Content: content, Language: "en",
		CrawledAt: time.Unix(0, 0).UTC(),
	}
}

func TestLookupYieldsAPageThatHoldsEveryWord(t *testing.T) {
	indexes := openIndexes(t)
	indexed(t, indexes,
		englishPage("https://example.com/guide", "Riverside Guide",
			"wildflowers grow along riverside trails"),
		englishPage("https://example.com/other", "Mountain Guide", "rocks and snow"),
	)

	matches := matchesFor(t, indexes, textsearch.Search{Query: "wildflower trails"})

	if len(matches) != 1 {
		t.Fatalf("matches = %+v", matches)
	}
	if matches[0].URL != "https://example.com/guide" || matches[0].Title != "Riverside Guide" {
		t.Errorf("match = %+v", matches[0])
	}
	if !strings.Contains(matches[0].Snippet, "wildflowers") {
		t.Errorf("snippet = %q", matches[0].Snippet)
	}
}

func TestLookupMatchesWordsSpreadOverTheFields(t *testing.T) {
	indexes := openIndexes(t)
	indexed(t, indexes, searchdocument.Document{
		URL: "https://example.com/spread", Title: "Riverside", Content: "trails",
		AuxiliaryText: "footnote", Language: "en",
	})

	matches := matchesFor(t, indexes, textsearch.Search{Query: "riverside trails footnote"})

	if got := urlsOf(matches); len(got) != 1 || got[0] != "https://example.com/spread" {
		t.Errorf("matches = %v", got)
	}
}

func TestLookupLeavesOutAPageMissingAWord(t *testing.T) {
	indexes := openIndexes(t)
	indexed(t, indexes, englishPage("https://example.com/guide", "Guide", "riverside trails"))

	if matches := matchesFor(t, indexes, textsearch.Search{Query: "riverside snow"}); len(
		matches,
	) != 0 {
		t.Errorf("matches = %+v", matches)
	}
}

func TestLookupRanksTheTitleAboveTheContent(t *testing.T) {
	indexes := openIndexes(t)
	indexed(t, indexes,
		englishPage("https://example.com/content", "Guide", "a lake of herons"),
		englishPage("https://example.com/title", "Herons", "a lake guide"),
	)

	got := urlsOf(matchesFor(t, indexes, textsearch.Search{Query: "herons"}))

	if len(got) != 2 || got[0] != "https://example.com/title" {
		t.Errorf("matches = %v, want the title match first", got)
	}
}

func TestLookupKeepsToTheSearchLanguage(t *testing.T) {
	indexes := openIndexes(t)
	indexed(t, indexes,
		englishPage("https://example.com/en", "Garden", "garden flowers"),
		searchdocument.Document{
			URL: "https://example.de/de", Title: "Garten", Content: "garden blumen",
			Language: "de",
		},
	)

	german := urlsOf(matchesFor(t, indexes, textsearch.Search{Query: "garden", Language: "de"}))
	every := urlsOf(matchesFor(t, indexes, textsearch.Search{Query: "garden"}))

	if len(german) != 1 || german[0] != "https://example.de/de" {
		t.Errorf("german matches = %v", german)
	}
	if len(every) != 2 {
		t.Errorf("matches in every language = %v", every)
	}
}

func TestLookupFindsAPageOfAnUnconfiguredLanguage(t *testing.T) {
	indexes := openIndexes(t)
	indexed(t, indexes, searchdocument.Document{
		URL: "https://example.es/flores", Title: "Jardín", Content: "flores del jardín",
		Language: "es",
	})

	got := urlsOf(matchesFor(t, indexes, textsearch.Search{Query: "jardin"}))

	if len(got) != 1 || got[0] != "https://example.es/flores" {
		t.Errorf("matches = %v", got)
	}
}

func TestIndexReplacesThePageOfTheSameURL(t *testing.T) {
	indexes := openIndexes(t)
	indexed(t, indexes,
		englishPage("https://example.com/guide", "Old Guide", "riverside"),
		englishPage("https://example.com/guide", "New Guide", "riverside"),
	)

	matches := matchesFor(t, indexes, textsearch.Search{Query: "riverside"})

	if len(matches) != 1 || matches[0].Title != "New Guide" {
		t.Errorf("matches = %+v", matches)
	}
}

func TestLookupCutsTheSnippet(t *testing.T) {
	indexes := openIndexes(t)
	longContent := strings.Repeat("meadow grass ", 100) + "heron " + strings.Repeat("reed ", 100)
	indexed(t, indexes,
		englishPage("https://example.com/long", "Long", longContent),
		englishPage("https://example.com/titled", "Heron Notes", longContent),
	)

	for _, match := range matchesFor(t, indexes, textsearch.Search{Query: "heron"}) {
		if utf8.RuneCountInString(match.Snippet) > textsearch.SnippetLength {
			t.Errorf("snippet of %s holds %d characters", match.URL,
				utf8.RuneCountInString(match.Snippet))
		}
		if !strings.Contains(match.Snippet, "heron") {
			t.Errorf("snippet of %s = %q", match.URL, match.Snippet)
		}
	}
}

func TestLookupStartsTheSnippetAtTheContentWhenOnlyTheTitleMatches(t *testing.T) {
	indexes := openIndexes(t)
	longContent := strings.Repeat("reed ", 100)
	indexed(t, indexes, englishPage("https://example.com/titled", "Heron", longContent))

	matches := matchesFor(t, indexes, textsearch.Search{Query: "heron"})

	if len(matches) != 1 {
		t.Fatalf("matches = %+v", matches)
	}
	want := longContent[:textsearch.SnippetLength]
	if matches[0].Snippet != want {
		t.Errorf("snippet = %q, want %q", matches[0].Snippet, want)
	}
}

func TestLookupYieldsThePageOfMatchesAskedFor(t *testing.T) {
	indexes := openIndexes(t)
	indexed(t, indexes,
		englishPage("https://example.com/1", "One", "heron"),
		englishPage("https://example.com/2", "Two", "heron"),
		englishPage("https://example.com/3", "Three", "heron"),
	)

	first := urlsOf(matchesFor(t, indexes, textsearch.Search{Query: "heron", Limit: 2}))
	rest := urlsOf(matchesFor(t, indexes, textsearch.Search{Query: "heron", Offset: 2, Limit: 2}))

	if len(first) != 2 || len(rest) != 1 {
		t.Fatalf("first page = %v, rest = %v", first, rest)
	}
	for _, url := range first {
		if url == rest[0] {
			t.Errorf("%s is on both pages", url)
		}
	}
}

func TestIndexesKeepThePagesAcrossOpenings(t *testing.T) {
	directory := t.TempDir()
	first := bootstrappedIndexes(t, directory)
	indexed(t, first, englishPage("https://example.com/guide", "Guide", "riverside"))
	first.Close(context.Background())

	reopened := bootstrappedIndexes(t, directory)
	t.Cleanup(func() { reopened.Close(context.Background()) })

	if got := urlsOf(matchesFor(t, reopened, textsearch.Search{Query: "riverside"})); len(
		got,
	) != 1 {
		t.Errorf("matches after reopening = %v", got)
	}
}

func TestIndexesFailWhenTheDirectoryCannotBeCreated(t *testing.T) {
	regularFile := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(regularFile, nil, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	languageIndexes, err := languageindex.IndexesFor("yacy_text", nil)
	if err != nil {
		t.Fatalf("language indexes: %v", err)
	}

	indexes := bleve.NewIndexes(filepath.Join(regularFile, "search-index"), languageIndexes)
	if err := indexes.Bootstrap(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
}
