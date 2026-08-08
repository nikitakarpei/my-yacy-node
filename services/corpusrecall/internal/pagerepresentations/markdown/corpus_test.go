package markdown_test

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

const (
	crawledURL      = "https://example.com/"
	crawledMarkdown = "# Hi"
	responseLimit   = 1024
)

func storedMarkdown(t *testing.T, markdownOfPage string) *markdown.Corpus {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	store, err := js.CreateOrUpdateObjectStore(context.Background(), jetstream.ObjectStoreConfig{
		Bucket: pagemarkdownstore.BucketName,
	})
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
	return markdown.NewCorpus(store, responseLimit)
}

func recalledPage(t *testing.T, corpus *markdown.Corpus) recall.Representation {
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
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	store, err := js.CreateOrUpdateObjectStore(context.Background(), jetstream.ObjectStoreConfig{
		Bucket: pagemarkdownstore.BucketName,
	})
	if err != nil {
		t.Fatalf("create page markdown bucket: %v", err)
	}
	if _, err := store.PutBytes(
		context.Background(), pagemarkdownstore.ObjectName(crawledURL), []byte(crawledMarkdown),
	); err != nil {
		t.Fatalf("store markdown: %v", err)
	}

	_, _, err = markdown.NewCorpus(store, 1).RepresentationOf(context.Background(), crawledURL)

	if err == nil {
		t.Fatal("expected an error for markdown beyond the response limit")
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
