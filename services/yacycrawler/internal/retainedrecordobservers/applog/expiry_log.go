// Package applog reports to the service log the crawl state records the sweep forgot
// once their retention ended, and the sweeps the storage refused.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgExpiredRecordsForgotten    = "expired crawl state records forgotten"
	msgExpiredRecordsNotForgotten = "expired crawl state records not forgotten, the next sweep tries again"
)

type ExpiryLog struct{}

func (ExpiryLog) ExpiredRecordsForgotten(ctx context.Context, bucket string, amount int) {
	slog.DebugContext(ctx, msgExpiredRecordsForgotten,
		slog.String("bucket", bucket),
		slog.Int("records", amount),
	)
}

func (ExpiryLog) ExpiredRecordsNotForgotten(ctx context.Context, bucket string, cause error) {
	slog.WarnContext(ctx, msgExpiredRecordsNotForgotten,
		slog.String("bucket", bucket),
		slog.Any("error", cause),
	)
}
