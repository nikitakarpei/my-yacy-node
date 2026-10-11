package s3_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	pagemarkdowncorporas3 "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/s3"
)

const (
	crawledURL = "https://example.com/"
	bucket     = "yacy-page-markdown"
)

var writeTime = time.Date(2026, 10, 12, 8, 30, 0, 123, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return writeTime }

func objectStoreServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(gofakes3.New(s3mem.New()).Server())
	t.Cleanup(server.Close)
	return server
}

func objectStoreAt(endpoint string) pagemarkdowncorporas3.ObjectStore {
	return pagemarkdowncorporas3.ObjectStore{
		Endpoint:        endpoint,
		Region:          "us-east-1",
		AccessKeyID:     "corpus",
		SecretAccessKey: "corpus-secret",
	}
}

func openedCorpus(t *testing.T, endpoint string) *pagemarkdowncorporas3.Corpus {
	t.Helper()
	corpus, err := pagemarkdowncorporas3.OpenCorpus(
		context.Background(), objectStoreAt(endpoint), bucket, fixedClock{},
	)
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	return corpus
}

func put(t *testing.T, corpus *pagemarkdowncorporas3.Corpus, markdown string) {
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
	corpus *pagemarkdowncorporas3.Corpus,
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
	corpus := openedCorpus(t, objectStoreServer(t).URL)
	put(t, corpus, "# Hi")

	stored := heldMarkdown(t, corpus)

	if string(stored.Markdown) != "# Hi" {
		t.Errorf("markdown = %q, want %q", stored.Markdown, "# Hi")
	}
	if !stored.StoredAt.Equal(writeTime) {
		t.Errorf("storedAt = %s, want the write time %s", stored.StoredAt, writeTime)
	}
	if stored.Version == "" {
		t.Error("version is empty, want a version naming the markdown held")
	}
}

func TestPutReplacesTheMarkdownOfARecrawledURL(t *testing.T) {
	corpus := openedCorpus(t, objectStoreServer(t).URL)
	put(t, corpus, "# Hi")
	put(t, corpus, "# Hi again")

	if got := string(heldMarkdown(t, corpus).Markdown); got != "# Hi again" {
		t.Errorf("markdown = %q, want %q", got, "# Hi again")
	}
}

func TestMarkdownOfVersionsTheMarkdownItYields(t *testing.T) {
	corpus := openedCorpus(t, objectStoreServer(t).URL)
	put(t, corpus, "# Hi")
	first := heldMarkdown(t, corpus)

	put(t, corpus, "# Hi")
	if unchanged := heldMarkdown(t, corpus); unchanged.Version != first.Version {
		t.Errorf(
			"version = %q after storing the same markdown, want the earlier %q",
			unchanged.Version, first.Version,
		)
	}

	put(t, corpus, "# Hi again")
	if changed := heldMarkdown(t, corpus); changed.Version == first.Version {
		t.Errorf("version = %q after storing other markdown, want a different one", changed.Version)
	}
}

func TestMarkdownOfReportsAURLTheCorpusDoesNotHold(t *testing.T) {
	corpus := openedCorpus(t, objectStoreServer(t).URL)

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
	endpoint := objectStoreServer(t).URL
	put(t, openedCorpus(t, endpoint), "# Hi")

	if got := string(heldMarkdown(t, openedCorpus(t, endpoint)).Markdown); got != "# Hi" {
		t.Errorf("markdown = %q, want %q", got, "# Hi")
	}
}

func TestOpenCorpusFailsWhenTheBucketCannotBeCreated(t *testing.T) {
	server := objectStoreServer(t)
	server.Close()

	if _, err := pagemarkdowncorporas3.OpenCorpus(
		context.Background(), objectStoreAt(server.URL), bucket, fixedClock{},
	); err == nil {
		t.Fatal("expected error when the page markdown bucket cannot be created")
	}
}

func TestPutFailsWhenTheMarkdownCannotBeWritten(t *testing.T) {
	server := objectStoreServer(t)
	corpus := openedCorpus(t, server.URL)
	server.Close()

	if err := corpus.Put(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, crawledURL),
		[]byte("# Hi"),
	); err == nil {
		t.Fatal("expected error when the markdown cannot be written")
	}
}

func TestMarkdownOfFailsWhenTheMarkdownCannotBeRead(t *testing.T) {
	server := objectStoreServer(t)
	corpus := openedCorpus(t, server.URL)
	server.Close()

	if _, _, err := corpus.MarkdownOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, crawledURL),
	); err == nil {
		t.Fatal("expected error when the markdown cannot be read")
	}
}
