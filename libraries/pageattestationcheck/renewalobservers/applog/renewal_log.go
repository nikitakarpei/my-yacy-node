// Package applog writes each failure to renew the page attestation check to the application log.
package applog

import (
	"context"
	"log/slog"
)

const msgTrustUnloaded = "revocation list of attestation keys not loaded"

type RenewalLog struct{}

func (RenewalLog) TrustUnloaded(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgTrustUnloaded, slog.Any("error", cause))
}
