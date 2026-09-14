package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgLinkDiscoveryRefusalEnforced = "page link discovery refusal enforced"

type RefusalEnforcementLog struct{}

func (RefusalEnforcementLog) LinkDiscoveryRefusalEnforced(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgLinkDiscoveryRefusalEnforced, slog.String("url", pageURL.String()))
}
