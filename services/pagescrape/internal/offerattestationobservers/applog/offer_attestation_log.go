// Package applog writes each page offer that goes without attestation to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgOfferNotAttested = "page offered without attestation"

type OfferAttestationLog struct{}

func (OfferAttestationLog) OfferNotAttested(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	reason string,
) {
	slog.WarnContext(ctx, msgOfferNotAttested,
		slog.String("pageUrl", pageURL.String()),
		slog.String("reason", reason),
	)
}
