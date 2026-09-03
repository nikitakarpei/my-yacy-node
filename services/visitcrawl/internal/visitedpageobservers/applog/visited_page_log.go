package applog

import (
	"context"
	"log/slog"
)

const (
	msgVisitedPageRedirected    = "visited page redirected"
	msgVisitedPageRejected      = "visited page rejected"
	msgVisitedPageMethodRefused = "visited page request method refused"
)

type VisitedPageLog struct{}

func (VisitedPageLog) VisitedPageRedirected(ctx context.Context, visitedPage string) {
	slog.DebugContext(ctx, msgVisitedPageRedirected, slog.String("visitedPage", visitedPage))
}

func (VisitedPageLog) VisitedPageRejected(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgVisitedPageRejected, slog.Any("error", cause))
}

func (VisitedPageLog) VisitedPageMethodRefused(ctx context.Context, method string) {
	slog.WarnContext(ctx, msgVisitedPageMethodRefused, slog.String("method", method))
}
