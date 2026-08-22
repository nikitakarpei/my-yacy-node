package jetstream_test

import (
	"context"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/disposedpages/jetstream"
)

func disposedPagesUnderTest(t *testing.T) *jetstream.DisposedPages {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateKeyValue(context.Background(), natsjetstream.KeyValueConfig{
		Bucket: yacycrawlcontract.DisposedPagesBucketName,
	}); err != nil {
		t.Fatal(err)
	}
	bucket, err := js.KeyValue(context.Background(), yacycrawlcontract.DisposedPagesBucketName)
	if err != nil {
		t.Fatal(err)
	}
	return jetstream.NewDisposedPages(bucket)
}

func TestDisposedPageOfYieldsTheReasonTheRecordCarries(t *testing.T) {
	disposedPages := disposedPagesUnderTest(t)
	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	if err := disposedPages.Record(context.Background(), url, disposal.Oversized); err != nil {
		t.Fatalf("record: %v", err)
	}

	disposedPage, disposed, err := disposedPages.DisposedPageOf(context.Background(), url)
	if err != nil {
		t.Fatalf("disposed page of: %v", err)
	}

	if !disposed {
		t.Fatal("expected the page to be disposed")
	}
	if disposedPage.Reason != disposal.Oversized {
		t.Fatalf("reason = %q, want %q", disposedPage.Reason, disposal.Oversized)
	}
}

func TestDisposedPageOfReportsAURLThatWasNeverDisposed(t *testing.T) {
	disposedPages := disposedPagesUnderTest(t)
	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")

	_, disposed, err := disposedPages.DisposedPageOf(context.Background(), url)
	if err != nil {
		t.Fatalf("disposed page of: %v", err)
	}

	if disposed {
		t.Fatal("expected the page not to be disposed")
	}
}

func TestDisposedPageOfYieldsAHigherMarkAfterEachDisposal(t *testing.T) {
	disposedPages := disposedPagesUnderTest(t)
	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	if err := disposedPages.Record(context.Background(), url, disposal.Oversized); err != nil {
		t.Fatalf("record: %v", err)
	}
	first, _, err := disposedPages.DisposedPageOf(context.Background(), url)
	if err != nil {
		t.Fatalf("disposed page of: %v", err)
	}

	if err := disposedPages.Record(context.Background(), url, disposal.Unextractable); err != nil {
		t.Fatalf("record again: %v", err)
	}

	second, _, err := disposedPages.DisposedPageOf(context.Background(), url)
	if err != nil {
		t.Fatalf("disposed page of: %v", err)
	}
	if second.Mark <= first.Mark {
		t.Fatalf("mark = %q, want it above %q", second.Mark, first.Mark)
	}
}
