package disposal_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

type recordingObserver struct {
	reasons []disposal.Reason
}

func (o *recordingObserver) PageDisposed(reason disposal.Reason) {
	o.reasons = append(o.reasons, reason)
}

type recordingDisposedPages struct {
	urls     []string
	failWith error
}

func (d *recordingDisposedPages) Record(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) error {
	d.urls = append(d.urls, canonicalURL.String())
	return d.failWith
}

func TestDisposeObservesAndRecordsTogether(t *testing.T) {
	observer := &recordingObserver{}
	disposed := &recordingDisposedPages{}

	disposal.NewDisposer(observer, disposed).
		Dispose(context.Background(), canonicalurltest.CanonicalURLOf(t, "http://host/a"), disposal.NotAPage)

	if len(observer.reasons) != 1 || observer.reasons[0] != disposal.NotAPage {
		t.Fatalf("observed reasons = %v", observer.reasons)
	}
	if len(disposed.urls) != 1 || disposed.urls[0] != "http://host/a" {
		t.Fatalf("recorded urls = %v", disposed.urls)
	}
}

func TestDisposeSurvivesRecordFailure(t *testing.T) {
	observer := &recordingObserver{}
	disposed := &recordingDisposedPages{failWith: errors.New("bucket down")}

	disposal.NewDisposer(observer, disposed).
		Dispose(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://host/a"),
			disposal.BudgetTruncated,
		)

	if len(observer.reasons) != 1 {
		t.Fatalf("a failed record must not hide the disposal, got %v", observer.reasons)
	}
}
