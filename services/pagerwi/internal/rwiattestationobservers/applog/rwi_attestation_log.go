// Package applog writes each RWI that goes without a page attestation to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgRWINotAttested = "rwi published without a page attestation"

type RWIAttestationLog struct{}

func (RWIAttestationLog) RWINotAttested(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	reason string,
) {
	slog.WarnContext(ctx, msgRWINotAttested,
		slog.String("pageUrl", pageURL.String()),
		slog.String("reason", reason),
	)
}
