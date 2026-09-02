// Package processrestartinterval holds a failed process until the shortest
// interval between two starts of it has passed. A supervisor that restarts a
// failed process without a delay of its own cannot then start it again sooner.
// A process that already ran longer than the interval exits at once.
package processrestartinterval

import (
	"context"
	"log/slog"
	"time"
)

func HoldTheExit(
	ctx context.Context,
	processStart time.Time,
	shortestInterval time.Duration,
) {
	remainder := shortestInterval - time.Since(processStart)
	if remainder <= 0 {
		return
	}

	slog.DebugContext(ctx, "exit held until the next start is due",
		slog.Duration("remainder", remainder),
	)

	held := time.NewTimer(remainder)
	defer held.Stop()

	select {
	case <-ctx.Done():
	case <-held.C:
	}
}
