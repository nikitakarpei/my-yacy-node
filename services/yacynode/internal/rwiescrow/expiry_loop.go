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

// TECHDEBT: testing time — waits on a real ticker with no clock port, so no test reaches the tick branch.
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
	var total int
	for ctx.Err() == nil {
		expired, err := expiry.Expire(ctx, config.HoldFor, config.Batch)
		if err != nil {
			observer.ObserveExpiryFailure()
			slog.ErrorContext(ctx, failedMessage, slog.Any("error", err))

			return
		}
		observer.ObserveExpired(expired)
		total += expired
		if expired < config.Batch {
			break
		}
	}
	if total == 0 {
		return
	}
	slog.DebugContext(ctx, expiredMessage, slog.Int("postings", total))
}
