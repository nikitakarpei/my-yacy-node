// Package applog writes each failure to renew the page attestation check to the application log.
package applog

import (
	"context"
	"log/slog"
)

const msgAndroidTrustUnloaded = "Android trust not loaded"

type RenewalLog struct{}

func (RenewalLog) AndroidTrustUnloaded(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgAndroidTrustUnloaded, slog.Any("error", cause))
}
