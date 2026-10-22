package vault

import (
	"context"
	"time"
)

type ExpiryObservers []ExpiryObserver

func (observers ExpiryObservers) ExpiryPassBegan(ctx context.Context, bucket Name, at time.Time) {
	for _, observer := range observers {
		observer.ExpiryPassBegan(ctx, bucket, at)
	}
}

func (observers ExpiryObservers) ExpiryPassFinished(
	ctx context.Context,
	bucket Name,
	at time.Time,
) {
	for _, observer := range observers {
		observer.ExpiryPassFinished(ctx, bucket, at)
	}
}

func (observers ExpiryObservers) ExpiredRecordsForgotten(
	ctx context.Context,
	bucket Name,
	amount int,
) {
	for _, observer := range observers {
		observer.ExpiredRecordsForgotten(ctx, bucket, amount)
	}
}

func (observers ExpiryObservers) ExpiredRecordsNotForgotten(
	ctx context.Context,
	bucket Name,
	cause error,
) {
	for _, observer := range observers {
		observer.ExpiredRecordsNotForgotten(ctx, bucket, cause)
	}
}
