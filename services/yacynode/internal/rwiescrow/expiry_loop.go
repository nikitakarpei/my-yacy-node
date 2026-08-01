package rwiescrow

import (
	"context"
	"log/slog"
	"time"
)

const (
	expiredMessage = "held postings expired"
	failedMessage  = "held posting expiry failed"
)

func RunExpiryLoop(
	ctx context.Context,
	expiry PostingExpiry,
	observer ExpiryObserver,
	config ExpiryConfig,
) {
	expireOnce(ctx, expiry, observer, config)

	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expireOnce(ctx, expiry, observer, config)
		}
	}
}

func expireOnce(
	ctx context.Context,
	expiry PostingExpiry,
	observer ExpiryObserver,
	config ExpiryConfig,
) {
	expired, err := expiry.Expire(ctx, config.HoldFor, config.Batch)
	if err != nil {
		observer.ObserveExpiryFailure()
		slog.ErrorContext(ctx, failedMessage, slog.Any("error", err))

		return
	}
	observer.ObserveExpired(expired)
	if expired == 0 {
		return
	}
	slog.DebugContext(ctx, expiredMessage, slog.Int("postings", expired))
}
