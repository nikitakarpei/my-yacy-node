// Package leadingword finds the lead of a query: the query word with the fewest
// documents in a partition, whose documents the other query words must match.
// Its document amounts give the amount of documents of each query word in one
// partition of the ring; a query has no lead when no word was counted.
package leadingword

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DocumentAsks interface {
	WhichDocumentsHaveIn(partition uint, words []yacymodel.Hash)
	WaitUntilPartitionSettledFor(partition uint, words []yacymodel.Hash)
	SettledIn(partition uint, words []yacymodel.Hash) documentasks.Answers
}

type DocumentAmounts interface {
	AmountsInAPartitionFor(
		ctx context.Context,
		query searchquery.Query,
		documentAsks DocumentAsks,
	) map[yacymodel.Hash]int
}

type Lead struct {
	Word                          yacymodel.Hash
	AmountOfDocumentsInAPartition int
}

type Finder struct {
	documentAmounts DocumentAmounts
}

func New(documentAmounts DocumentAmounts) Finder {
	return Finder{documentAmounts: documentAmounts}
}

func (finder Finder) FindFor(
	ctx context.Context,
	query searchquery.Query,
	documentAsks DocumentAsks,
) yacymodel.Optional[Lead] {
	return rarestAmong(
		query.WordHashes(),
		finder.documentAmounts.AmountsInAPartitionFor(ctx, query, documentAsks),
	)
}

func rarestAmong(
	queryWords []yacymodel.Hash,
	amountsInAPartition map[yacymodel.Hash]int,
) yacymodel.Optional[Lead] {
	rarestLead := yacymodel.None[Lead]()
	for _, word := range queryWords {
		amountOfDocuments, counted := amountsInAPartition[word]
		if !counted {
			continue
		}
		if lead, led := rarestLead.Get(); led &&
			amountOfDocuments >= lead.AmountOfDocumentsInAPartition {
			continue
		}
		rarestLead = yacymodel.Some(
			Lead{Word: word, AmountOfDocumentsInAPartition: amountOfDocuments},
		)
	}

	return rarestLead
}
