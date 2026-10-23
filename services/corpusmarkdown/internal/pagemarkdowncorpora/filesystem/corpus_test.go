package filesystem_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	pagemarkdowncorporafilesystem "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/filesystem"
)

const crawledURL = "https://example.com/"

var writeTime = time.Date(2026, 10, 12, 8, 30, 0, 123, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return writeTime }

func openedCorpus(t *testing.T, directory string) *pagemarkdowncorporafilesystem.Corpus {
	t.Helper()
	corpus, err := pagemarkdowncorporafilesystem.OpenCorpus(directory, fixedClock{})
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	return corpus
}

func put(t *testing.T, corpus *pagemarkdowncorporafilesystem.Corpus, markdown string) {
	t.Helper()
	if err := corpus.Put(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, crawledURL),
		[]byte(markdown),
	); err != nil {
		t.Fatalf("put %q: %v", markdown, err)
	}
}

func heldMarkdown(
	t *testing.T,
	corpus *pagemarkdowncorporafilesystem.Corpus,
) markdownrecall.StoredMarkdown {
	t.Helper()
	stored, held, err := corpus.MarkdownOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, crawledURL),
	)
	if err != nil {
		t.Fatalf("markdown of %q: %v", crawledURL, err)
	}
	if !held {
		t.Fatalf("corpus holds no markdown for %q", crawledURL)
	}
	return stored
}

func TestMarkdownOfYieldsTheMarkdownHeldUnderTheCanonicalURL(t *testing.T) {
	corpus := openedCorpus(t, t.TempDir())
	put(t, corpus, "# Hi")

	stored := heldMarkdown(t, corpus)

	if string(stored.Markdown) != "# Hi" {
		t.Errorf("markdown = %q, want %q", stored.Markdown, "# Hi")
	}
	if !stored.StoredAt.Equal(writeTime) {
		t.Errorf("storedAt = %s, want the write time %s", stored.StoredAt, writeTime)
	}
	if stored.Version != markdownrecall.VersionOf([]byte("# Hi")) {
		t.Errorf("version = %q, want the version of the markdown held", stored.Version)
	}
}

func TestPutReplacesTheMarkdownOfARecrawledURL(t *testing.T) {
	corpus := openedCorpus(t, t.TempDir())
	put(t, corpus, "# Hi")
	put(t, corpus, "# Hi again")

	if got := string(heldMarkdown(t, corpus).Markdown); got != "# Hi again" {
		t.Errorf("markdown = %q, want %q", got, "# Hi again")
	}
}

func TestMarkdownOfReportsAURLTheCorpusDoesNotHold(t *testing.T) {
	corpus := openedCorpus(t, t.TempDir())

	_, held, err := corpus.MarkdownOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, crawledURL),
	)
	if err != nil {
		t.Fatalf("markdown of %q: %v", crawledURL, err)
	}
	if held {
		t.Errorf("corpus reports markdown for %q it never held", crawledURL)
	}
}

func TestOpenCorpusKeepsTheMarkdownOfAnEarlierOpening(t *testing.T) {
	directory := t.TempDir()
	put(t, openedCorpus(t, directory), "# Hi")

	if got := string(heldMarkdown(t, openedCorpus(t, directory)).Markdown); got != "# Hi" {
		t.Errorf("markdown = %q, want %q", got, "# Hi")
	}
}

func TestOpenCorpusCreatesAMissingDirectory(t *testing.T) {
	corpus := openedCorpus(t, filepath.Join(t.TempDir(), "page", "markdown"))
	put(t, corpus, "# Hi")

	if got := string(heldMarkdown(t, corpus).Markdown); got != "# Hi" {
		t.Errorf("markdown = %q, want %q", got, "# Hi")
	}
}

func TestOpenCorpusFailsWhenTheDirectoryCannotBeCreated(t *testing.T) {
	occupied := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(occupied, nil, 0o600); err != nil {
		t.Fatalf("occupy the path: %v", err)
	}

	if _, err := pagemarkdowncorporafilesystem.OpenCorpus(
		filepath.Join(occupied, "markdown"), fixedClock{},
	); err == nil {
		t.Fatal("expected error when the page markdown directory cannot be created")
	}
}

func TestPutFailsWhenTheMarkdownCannotBeWritten(t *testing.T) {
	directory := t.TempDir()
	corpus := openedCorpus(t, directory)
	if err := os.RemoveAll(directory); err != nil {
		t.Fatalf("remove the page markdown directory: %v", err)
	}

	if err := corpus.Put(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, crawledURL),
		[]byte("# Hi"),
	); err == nil {
		t.Fatal("expected error when the markdown cannot be written")
	}
}
