// Package applog writes each failure to renew the page attestation verifier to the application
// log.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgAndroidTrustUnloaded = "Android trust not loaded"
	msgRenewalRefused       = "page attestation verifier not renewed, no row is attested"
)

type RenewalLog struct{}

func (RenewalLog) AndroidTrustUnloaded(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgAndroidTrustUnloaded, slog.Any("error", cause))
}

func (RenewalLog) RenewalRefused(ctx context.Context, reason error) {
	slog.WarnContext(ctx, msgRenewalRefused, slog.Any("reason", reason))
}
