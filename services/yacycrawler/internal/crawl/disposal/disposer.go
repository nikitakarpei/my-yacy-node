// Package disposal records a page that reached a terminal outcome without publication, and why.
package disposal

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const msgDisposedPageRecordFailed = "disposed page record failed, recall will wait out the deadline"

type DisposalProgress interface {
	PageDisposed(reason Reason)
}

type DisposedPages interface {
	Record(ctx context.Context, canonicalURL yacycrawlcontract.CanonicalURL) error
}

type Disposer struct {
	observer DisposalProgress
	disposed DisposedPages
}

func NewDisposer(observer DisposalProgress, disposed DisposedPages) *Disposer {
	return &Disposer{observer: observer, disposed: disposed}
}

func (d *Disposer) Dispose(
	ctx context.Context,
	canonicalURL yacycrawlcontract.CanonicalURL,
	reason Reason,
) {
	d.observer.PageDisposed(reason)
	if err := d.disposed.Record(ctx, canonicalURL); err != nil {
		slog.WarnContext(ctx, msgDisposedPageRecordFailed,
			slog.String("url", canonicalURL.String()),
			slog.Any("error", err),
		)
	}
}
