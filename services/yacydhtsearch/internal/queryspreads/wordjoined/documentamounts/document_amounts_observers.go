package documentamounts

import "context"

type PerformedFromReplicas struct {
	Partition                 uint
	AmountOfQueryWords        int
	AmountOfQueryWordsCounted int
}

type PerformedFromCache struct {
	AmountOfQueryWords       int
	AmountOfQueryWordsCached int
	AllQueryWordsCached      bool
}

type FromReplicasObserver interface {
	AmountsCountedFromReplicas(ctx context.Context, performed PerformedFromReplicas)
}

type FromReplicasObservers []FromReplicasObserver

func (observers FromReplicasObservers) AmountsCountedFromReplicas(
	ctx context.Context,
	performed PerformedFromReplicas,
) {
	for _, observer := range observers {
		observer.AmountsCountedFromReplicas(ctx, performed)
	}
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
