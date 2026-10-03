// Package leadingword finds the lead of a query: the query word with the fewest
// documents in a partition, whose documents the other query words must match.
// Its document amounts give the amount of documents of each query word in one
// partition of the ring; a word without an amount cannot lead.
package leadingword

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type WordAsks interface {
	AskPartitionFor(partition uint, words []yacymodel.Hash)
	WaitUntilPartitionSettledFor(partition uint, words []yacymodel.Hash)
	SettledIn(partition uint, words []yacymodel.Hash) []wordpartitionasks.SettledAsk
}

type DocumentAmounts interface {
	AmountsInAPartitionFor(
		ctx context.Context,
		query searchquery.Query,
		run WordAsks,
	) map[yacymodel.Hash]int
}

type Lead struct {
	Word                          yacymodel.Optional[yacymodel.Hash]
	AmountOfDocumentsInAPartition yacymodel.Optional[int]
}

type Finder struct {
	documentAmounts DocumentAmounts
}

func New(documentAmounts DocumentAmounts) Finder {
	return Finder{documentAmounts: documentAmounts}
}

func (finder Finder) FindFor(ctx context.Context, query searchquery.Query, run WordAsks) Lead {
	return rarestAmong(
		query.WordHashes(),
		finder.documentAmounts.AmountsInAPartitionFor(ctx, query, run),
	)
}

func rarestAmong(queryWords []yacymodel.Hash, amountsInAPartition map[yacymodel.Hash]int) Lead {
	lead := Lead{
		Word:                          yacymodel.None[yacymodel.Hash](),
		AmountOfDocumentsInAPartition: yacymodel.None[int](),
	}
	for _, word := range queryWords {
		amountOfDocuments, counted := amountsInAPartition[word]
		if !counted {
			continue
		}
		if fewestDocuments, led := lead.AmountOfDocumentsInAPartition.Get(); led &&
			amountOfDocuments >= fewestDocuments {
			continue
		}
		lead = Lead{
			Word:                          yacymodel.Some(word),
			AmountOfDocumentsInAPartition: yacymodel.Some(amountOfDocuments),
		}
	}

	return lead
}
