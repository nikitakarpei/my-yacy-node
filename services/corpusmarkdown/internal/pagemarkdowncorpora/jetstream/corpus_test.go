package jetstream_test

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	pagemarkdowncorporajetstream "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

const crawledURL = "https://example.com/"

func openedCorpus(t *testing.T, url string) *pagemarkdowncorporajetstream.Corpus {
	t.Helper()
	corpus, err := pagemarkdowncorporajetstream.OpenCorpus(
		context.Background(),
		natstestserver.ConnectJetStream(t, url),
	)
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	return corpus
}

func storedMarkdown(t *testing.T, url string, canonicalURL canonicalurl.CanonicalURL) string {
	t.Helper()
	objects, err := natstestserver.ConnectJetStream(t, url).
		ObjectStore(context.Background(), pagemarkdownstore.BucketName)
	if err != nil {
		t.Fatalf("open page markdown bucket: %v", err)
	}
	markdown, err := objects.GetBytes(
		context.Background(),
		pagemarkdownstore.ObjectName(canonicalURL),
	)
	if err != nil {
		t.Fatalf("get markdown for %q: %v", canonicalURL, err)
	}
	return string(markdown)
}

func bucketStatus(t *testing.T, url string) jetstream.ObjectStoreStatus {
	t.Helper()
	objects, err := natstestserver.ConnectJetStream(t, url).
		ObjectStore(context.Background(), pagemarkdownstore.BucketName)
	if err != nil {
		t.Fatalf("open page markdown bucket: %v", err)
	}
	status, err := objects.Status(context.Background())
	if err != nil {
		t.Fatalf("page markdown bucket status: %v", err)
	}
	return status
}

func TestOpenCorpusCompressesTheStoredMarkdown(t *testing.T) {
	url := natstestserver.Start(t)
	openedCorpus(t, url)

	if !bucketStatus(t, url).IsCompressed() {
		t.Error("page markdown bucket is not compressed")
	}
}

func TestPutHoldsTheMarkdownUnderTheCanonicalURL(t *testing.T) {
	url := natstestserver.Start(t)
	corpus := openedCorpus(t, url)

	if err := corpus.Put(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, crawledURL),
		[]byte("# Hi"),
	); err != nil {
		t.Fatalf("put: %v", err)
	}

	if got := storedMarkdown(
		t,
		url,
		canonicalurltest.CanonicalURLOf(t, crawledURL),
	); got != "# Hi" {
		t.Errorf("stored = %q, want %q", got, "# Hi")
	}
}

func TestPutReplacesTheMarkdownOfARecrawledURL(t *testing.T) {
	url := natstestserver.Start(t)
	corpus := openedCorpus(t, url)

	if err := corpus.Put(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, crawledURL),
		[]byte("# Hi"),
	); err != nil {
		t.Fatalf("first put: %v", err)
	}
	if err := corpus.Put(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, crawledURL),
		[]byte("# Hi again"),
	); err != nil {
		t.Fatalf("second put: %v", err)
	}

	if got := storedMarkdown(
		t,
		url,
		canonicalurltest.CanonicalURLOf(t, crawledURL),
	); got != "# Hi again" {
		t.Errorf("stored = %q, want %q", got, "# Hi again")
	}
}

func TestPutFailsWhenTheMarkdownCannotBeWritten(t *testing.T) {
	corpus := openedCorpus(t, natstestserver.Start(t))
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if err := corpus.Put(
		abandoned,
		canonicalurltest.CanonicalURLOf(t, crawledURL),
		[]byte("# Hi"),
	); err == nil {
		t.Fatal("expected error when the markdown cannot be written")
	}
}

func TestOpenCorpusFailsWhenTheBucketCannotBeCreated(t *testing.T) {
	pageMarkdownJetStream := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if _, err := pagemarkdowncorporajetstream.OpenCorpus(
		abandoned,
		pageMarkdownJetStream,
	); err == nil {
		t.Fatal("expected error when the page markdown bucket cannot be created")
	}
}
