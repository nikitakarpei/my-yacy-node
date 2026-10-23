package vault

import (
	"context"
	"log/slog"
	"time"
)

type ExpiryObserver interface {
	ExpiryPassBegan(ctx context.Context, bucket Name, at time.Time)
	ExpiryPassFinished(ctx context.Context, bucket Name, at time.Time)
	ExpiredRecordsForgotten(ctx context.Context, bucket Name, amount int)
	ExpiredRecordsNotForgotten(ctx context.Context, bucket Name, cause error)
}

type silentExpiryObserver struct{}

func (silentExpiryObserver) ExpiryPassBegan(context.Context, Name, time.Time) {}

func (silentExpiryObserver) ExpiryPassFinished(context.Context, Name, time.Time) {}

func (silentExpiryObserver) ExpiredRecordsForgotten(context.Context, Name, int) {}

func (silentExpiryObserver) ExpiredRecordsNotForgotten(context.Context, Name, error) {}

type expiryLog struct {
	observer ExpiryObserver
}

func (l expiryLog) ExpiryPassBegan(ctx context.Context, bucket Name, at time.Time) {
	l.observer.ExpiryPassBegan(ctx, bucket, at)
	slog.DebugContext(ctx, "vault pass over expired records began",
		slog.String("bucket", string(bucket)),
		slog.Time("at", at),
	)
}

func (l expiryLog) ExpiryPassFinished(ctx context.Context, bucket Name, at time.Time) {
	l.observer.ExpiryPassFinished(ctx, bucket, at)
	slog.DebugContext(ctx, "vault pass over expired records finished",
		slog.String("bucket", string(bucket)),
		slog.Time("at", at),
	)
}

func (l expiryLog) ExpiredRecordsForgotten(ctx context.Context, bucket Name, amount int) {
	l.observer.ExpiredRecordsForgotten(ctx, bucket, amount)
	slog.DebugContext(ctx, "vault forgot expired records",
		slog.String("bucket", string(bucket)),
		slog.Int("records", amount),
	)
}

func (l expiryLog) ExpiredRecordsNotForgotten(ctx context.Context, bucket Name, cause error) {
	l.observer.ExpiredRecordsNotForgotten(ctx, bucket, cause)
	slog.WarnContext(ctx, "vault could not forget expired records, the next pass tries again",
		slog.String("bucket", string(bucket)),
		slog.Any("error", cause),
	)
}
