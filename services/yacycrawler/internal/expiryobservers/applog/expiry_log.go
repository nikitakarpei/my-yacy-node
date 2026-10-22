// Package applog reports to the service log each pass that forgets expired crawl
// state records, the records it forgot, and the records it could not forget.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	msgExpiryPassBegan            = "pass over expired crawl state records began"
	msgExpiryPassFinished         = "pass over expired crawl state records finished"
	msgExpiredRecordsForgotten    = "expired crawl state records forgotten"
	msgExpiredRecordsNotForgotten = "expired crawl state records not forgotten, the next pass tries again"
)

type ExpiryLog struct{}

func (ExpiryLog) ExpiryPassBegan(ctx context.Context, bucket vault.Name, at time.Time) {
	slog.DebugContext(ctx, msgExpiryPassBegan,
		slog.String("bucket", string(bucket)),
		slog.Time("at", at),
	)
}

func (ExpiryLog) ExpiryPassFinished(ctx context.Context, bucket vault.Name, at time.Time) {
	slog.DebugContext(ctx, msgExpiryPassFinished,
		slog.String("bucket", string(bucket)),
		slog.Time("at", at),
	)
}

func (ExpiryLog) ExpiredRecordsForgotten(ctx context.Context, bucket vault.Name, amount int) {
	slog.DebugContext(ctx, msgExpiredRecordsForgotten,
		slog.String("bucket", string(bucket)),
		slog.Int("records", amount),
	)
}

func (ExpiryLog) ExpiredRecordsNotForgotten(ctx context.Context, bucket vault.Name, cause error) {
	slog.WarnContext(ctx, msgExpiredRecordsNotForgotten,
		slog.String("bucket", string(bucket)),
		slog.Any("error", cause),
	)
}
