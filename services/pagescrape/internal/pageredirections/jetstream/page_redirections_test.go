package jetstream_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	pageredirections "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/pageredirections/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	requestedPageURL = "https://example.org/a"
	landedPageURL    = "https://example.org/b"
)

func TestRecordedRedirectionIsHeldUnderTheRequestedURL(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	requestedURL := canonicalurltest.CanonicalURLOf(t, requestedPageURL)
	pageURL := canonicalurltest.CanonicalURLOf(t, landedPageURL)

	redirections, err := pageredirections.CreatePageRedirections(ctx, broker)
	if err != nil {
		t.Fatalf("create the page redirections: %v", err)
	}
	if err := redirections.Record(ctx, requestedURL, pageURL); err != nil {
		t.Fatalf("record the redirection: %v", err)
	}

	bucket, err := broker.KeyValue(ctx, pagescrapecontract.PageRedirectionsBucketName)
	if err != nil {
		t.Fatalf("open the redirections bucket: %v", err)
	}
	entry, err := bucket.Get(ctx, pagescrapecontract.PageRedirectionKeyOf(requestedURL))
	if err != nil {
		t.Fatalf("get the redirection of %q: %v", requestedURL, err)
	}
	if got := string(entry.Value()); got != pageURL.String() {
		t.Errorf("%s redirects to %s, want %s", requestedURL, got, pageURL)
	}
}
