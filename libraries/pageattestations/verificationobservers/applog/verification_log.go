// Package applog writes the failures of the page attestation verifier to the application log.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgAndroidTrustUnloaded = "Android trust not loaded"
	msgNoSignerTrusted      = "no app signer is trusted, no row is attested"
	msgRowNotAttested       = "row not attested"
)

type VerificationLog struct{}

func (VerificationLog) AndroidTrustUnloaded(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgAndroidTrustUnloaded, slog.Any("error", cause))
}

func (VerificationLog) NoSignerTrusted(ctx context.Context, reason error) {
	slog.WarnContext(ctx, msgNoSignerTrusted, slog.Any("reason", reason))
}

func (VerificationLog) RowNotAttested(ctx context.Context, address string, reason error) {
	slog.DebugContext(
		ctx,
		msgRowNotAttested,
		slog.String("address", address),
		slog.Any("reason", reason),
	)
}
