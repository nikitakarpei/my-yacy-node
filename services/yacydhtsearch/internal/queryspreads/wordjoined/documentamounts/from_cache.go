// Package documentamounts tells the amount of documents each query word has. Before
// the inquiry, it tells the amount in one partition of the ring that earlier
// spreads remembered, and tells none when a query word has no remembered amount.
// Its measurement of the inquiry tells, from every answer, the amount held in the
// whole network and in a partition.
package documentamounts

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type CachedDocumentAmounts interface {
	DocumentAmountsOf(ctx context.Context, words []yacymodel.Hash) map[yacymodel.Hash]int
}

type FromCache struct {
	cachedAmounts CachedDocumentAmounts
	observer      FromCacheObserver
}

func NewFromCache(cachedAmounts CachedDocumentAmounts, observer FromCacheObserver) FromCache {
	return FromCache{cachedAmounts: cachedAmounts, observer: observer}
}

func (fromCache FromCache) AmountsInAPartitionFor(
	ctx context.Context,
	query searchquery.Query,
) map[yacymodel.Hash]int {
	amountsInAPartition := fromCache.cachedAmounts.DocumentAmountsOf(ctx, query.WordHashes())
	allQueryWordsCached := len(amountsInAPartition) == len(query.WordHashes())
	fromCache.observer.AmountsReadFromCache(ctx, PerformedFromCache{
		AmountOfQueryWords:       len(query.WordHashes()),
		AmountOfQueryWordsCached: len(amountsInAPartition),
		AllQueryWordsCached:      allQueryWordsCached,
	})
	if !allQueryWordsCached {
		return nil
	}

	return amountsInAPartition
}
