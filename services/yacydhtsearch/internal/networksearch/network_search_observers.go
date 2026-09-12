package networksearch

import "context"

type NetworkSearchObservers []NetworkSearchObserver

func (observers NetworkSearchObservers) NetworkSearchPerformed(
	ctx context.Context,
	search PerformedNetworkSearch,
) {
	for _, observer := range observers {
		observer.NetworkSearchPerformed(ctx, search)
	}
}
