package documentamounts

import "context"

type PerformedFromCache struct {
	AmountOfQueryWords       int
	AmountOfQueryWordsCached int
	AllQueryWordsCached      bool
}

type FromCacheObserver interface {
	AmountsReadFromCache(ctx context.Context, performed PerformedFromCache)
}

type FromCacheObservers []FromCacheObserver

func (observers FromCacheObservers) AmountsReadFromCache(
	ctx context.Context,
	performed PerformedFromCache,
) {
	for _, observer := range observers {
		observer.AmountsReadFromCache(ctx, performed)
	}
}
