package documentamounts

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type CachedDocumentAmounts interface {
	DocumentAmountsOf(ctx context.Context, words []yacymodel.Hash) map[yacymodel.Hash]int
}

type DocumentAmounts interface {
	AmountsInAPartitionFor(
		ctx context.Context,
		query searchquery.Query,
		documentAsks leadingword.DocumentAsks,
	) map[yacymodel.Hash]int
}

type FromCache struct {
	cachedAmounts   CachedDocumentAmounts
	uncachedAmounts DocumentAmounts
	observer        FromCacheObserver
}

func NewFromCache(
	cachedAmounts CachedDocumentAmounts,
	uncachedAmounts DocumentAmounts,
	observer FromCacheObserver,
) FromCache {
	return FromCache{
		cachedAmounts:   cachedAmounts,
		uncachedAmounts: uncachedAmounts,
		observer:        observer,
	}
}

func (fromCache FromCache) AmountsInAPartitionFor(
	ctx context.Context,
	query searchquery.Query,
	documentAsks leadingword.DocumentAsks,
) map[yacymodel.Hash]int {
	amountsInAPartition := fromCache.cachedAmounts.DocumentAmountsOf(ctx, query.WordHashes())
	allQueryWordsCached := len(amountsInAPartition) == len(query.WordHashes())
	fromCache.observer.AmountsReadFromCache(ctx, PerformedFromCache{
		AmountOfQueryWords:       len(query.WordHashes()),
		AmountOfQueryWordsCached: len(amountsInAPartition),
		AllQueryWordsCached:      allQueryWordsCached,
	})
	if !allQueryWordsCached {
		return fromCache.uncachedAmounts.AmountsInAPartitionFor(ctx, query, documentAsks)
	}

	return amountsInAPartition
}
