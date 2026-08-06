// Package disposal records a page that reached a terminal outcome without publication, and why.
package disposal

import (
	"context"
	"log/slog"
)

const msgDisposedPageRecordFailed = "disposed page record failed, recall will wait out the deadline"

type DisposalProgress interface {
	PageDisposed(reason Reason)
}

type DisposedPages interface {
	Record(ctx context.Context, url string) error
}

type Disposer struct {
	observer DisposalProgress
	disposed DisposedPages
}

func NewDisposer(observer DisposalProgress, disposed DisposedPages) *Disposer {
	return &Disposer{observer: observer, disposed: disposed}
}

func (d *Disposer) Dispose(ctx context.Context, url string, reason Reason) {
	d.observer.PageDisposed(reason)
	if err := d.disposed.Record(ctx, url); err != nil {
		slog.WarnContext(ctx, msgDisposedPageRecordFailed,
			slog.String("url", url),
			slog.Any("error", err),
		)
	}
}
