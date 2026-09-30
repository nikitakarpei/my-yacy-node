// Package applog writes the requests refused for their response headers
// deadline to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/deadlineenforcing"
)

type DeadlineLog struct{}

func (DeadlineLog) RequestRefused(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason deadlineenforcing.RefusalReason,
) {
	slog.WarnContext(ctx, "request refused",
		slog.String("address", address.String()),
		slog.String("reason", string(reason)),
	)
}
