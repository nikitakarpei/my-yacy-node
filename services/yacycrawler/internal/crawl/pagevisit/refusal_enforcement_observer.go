package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type RefusalEnforcementObserver interface {
	LinkDiscoveryRefusalEnforced(ctx context.Context, pageURL canonicalurl.CanonicalURL)
}

type RefusalEnforcementObservers []RefusalEnforcementObserver

func (observers RefusalEnforcementObservers) LinkDiscoveryRefusalEnforced(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.LinkDiscoveryRefusalEnforced(ctx, pageURL)
	}
}
