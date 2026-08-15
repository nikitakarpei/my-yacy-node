package jetstream_test

import (
	"context"
	"runtime"
	"strings"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	markdownjetstream "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

const (
	crawledURL      = "https://example.com/"
	crawledMarkdown = "# Hi"
	responseLimit   = 1024
)

func storedMarkdown(t *testing.T, markdownOfPage string) *markdownjetstream.Corpus {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	store, err := js.CreateOrUpdateObjectStore(
		context.Background(),
		natsjetstream.ObjectStoreConfig{
			Bucket: pagemarkdownstore.BucketName,
		},
	)
	if err != nil {
		t.Fatalf("create page markdown bucket: %v", err)
	}
	if markdownOfPage != "" {
		if _, err := store.PutBytes(
			context.Background(),
			pagemarkdownstore.ObjectName(crawledURL),
			[]byte(markdownOfPage),
		); err != nil {
			t.Fatalf("store markdown: %v", err)
		}
	}
	return markdownjetstream.NewCorpus(store, responseLimit)
}

func recalledPage(t *testing.T, corpus *markdownjetstream.Corpus) recall.Representation {
	t.Helper()
	representation, found, err := corpus.RepresentationOf(context.Background(), crawledURL)
	if err != nil {
		t.Fatalf("representation of %q: %v", crawledURL, err)
	}
	if !found {
		t.Fatalf("corpus holds no markdown for %q", crawledURL)
	}
	return representation
}

func TestRepresentationIsTheMarkdownPageTheCorpusHolds(t *testing.T) {
	representation := recalledPage(t, storedMarkdown(t, crawledMarkdown))

	if representation != (markdown.Page{CanonicalURL: crawledURL, Markdown: crawledMarkdown}) {
		t.Errorf("representation = %+v", representation)
	}
}

func TestRepresentationIsMissingWhenTheCorpusHoldsNoMarkdown(t *testing.T) {
	corpus := storedMarkdown(t, "")

	representation, found, err := corpus.RepresentationOf(context.Background(), crawledURL)
	if err != nil {
		t.Fatalf("representation of %q: %v", crawledURL, err)
	}
	if found || representation != nil {
		t.Errorf("found=%v representation=%v, want absent", found, representation)
	}
}

func TestRepresentationFailsWhenTheMarkdownExceedsTheResponseLimit(t *testing.T) {
	corpus := storedMarkdown(t, strings.Repeat("a", responseLimit+1))

	if _, _, err := corpus.RepresentationOf(context.Background(), crawledURL); err == nil {
		t.Fatal("expected an error for markdown beyond the response limit")
	}
}

func TestMarkdownBeyondTheResponseLimitIsNeverHeldInMemory(t *testing.T) {
	const oversizedBytes = 8 << 20
	corpus := storedMarkdown(t, strings.Repeat("a", oversizedBytes))

	var beforeRecall, afterRecall runtime.MemStats
	runtime.ReadMemStats(&beforeRecall)
	_, _, err := corpus.RepresentationOf(context.Background(), crawledURL)
	runtime.ReadMemStats(&afterRecall)

	if err == nil {
		t.Fatal("expected an error for markdown beyond the response limit")
	}
	allocated := afterRecall.TotalAlloc - beforeRecall.TotalAlloc
	if allocated > oversizedBytes/4 {
		t.Fatalf(
			"refusing %d bytes of markdown allocated %d bytes, want far less",
			oversizedBytes, allocated,
		)
	}
}

func TestRepresentationFailsWhenTheCorpusCannotBeRead(t *testing.T) {
	corpus := storedMarkdown(t, crawledMarkdown)
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if _, _, err := corpus.RepresentationOf(abandoned, crawledURL); err == nil {
		t.Fatal("expected an error when the corpus cannot be read")
	}
}
