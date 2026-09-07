// Package applog reports what each configured seedlist published.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgSeedlistRead        = "seedlist read"
	msgSeedlistUnreachable = "seedlist could not be reached"
	msgSeedlistUnreadable  = "seedlist could not be read"
)

type SeedlistLog struct{}

func (SeedlistLog) SeedlistRead(ctx context.Context, address string, seeds int) {
	slog.DebugContext(ctx, msgSeedlistRead,
		slog.String("address", address),
		slog.Int("seeds", seeds),
	)
}

func (SeedlistLog) SeedlistUnreachable(ctx context.Context, address string, cause error) {
	slog.WarnContext(ctx, msgSeedlistUnreachable,
		slog.String("address", address),
		slog.Any("error", cause),
	)
}

func (SeedlistLog) SeedlistUnreadable(ctx context.Context, address string, cause error) {
	slog.ErrorContext(ctx, msgSeedlistUnreadable,
		slog.String("address", address),
		slog.Any("error", cause),
	)
}
