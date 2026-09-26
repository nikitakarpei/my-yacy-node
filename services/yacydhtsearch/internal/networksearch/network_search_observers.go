package networksearch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type NetworkSearchObservers []NetworkSearchObserver

func (observers NetworkSearchObservers) NetworkSearchPerformed(
	ctx context.Context,
	search PerformedNetworkSearch,
) {
	for _, observer := range observers {
		observer.NetworkSearchPerformed(ctx, search)
	}
}

func (observers NetworkSearchObservers) QueryReachedNoPeer(
	ctx context.Context,
	query searchquery.Query,
) {
	for _, observer := range observers {
		observer.QueryReachedNoPeer(ctx, query)
	}
}
