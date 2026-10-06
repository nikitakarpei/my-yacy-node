// Package wordjoined finds documents that match the whole query when no single
// peer holds every query word, by joining what the peers of each word hold.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentjoin"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentsperword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordroles"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type RememberedDocumentAmounts interface {
	Remember(ctx context.Context, documentAmounts map[yacymodel.Hash]int)
}

type Spread struct {
	documentsAsker       documentasks.Asker
	documentAmountsCache RememberedDocumentAmounts
	leadingWordFinder    leadingword.Finder
	urlMetadataAsker     urlmetadataasks.Asker
	partitions           yacymodel.DHTRingPartitions
	observer             WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the spread takes each unit it asks through and each setting
func New(
	documentsAsker documentasks.Asker,
	documentAmountsCache RememberedDocumentAmounts,
	leadingWordFinder leadingword.Finder,
	urlMetadataAsker urlmetadataasks.Asker,
	partitions yacymodel.DHTRingPartitions,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		documentsAsker:       documentsAsker,
		documentAmountsCache: documentAmountsCache,
		leadingWordFinder:    leadingWordFinder,
		urlMetadataAsker:     urlMetadataAsker,
		partitions:           partitions,
		observer:             observer,
	}
}

func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryfindings.Findings {
	startedAt := time.Now()
	holders := documentholders.NoneYet()
	urlMetadataLookup := spread.urlMetadataAsker.Begin(ctx)
	documentsJoiner := documentjoin.JoinerOf(query, documentjoin.JoinObservers{
		urlMetadataLookupOfJoinedDocuments{holders, urlMetadataLookup},
	})
	documentInquiry := spread.documentsAsker.Begin(
		ctx, query, chosenPeersPerQueryWord,
		documentasks.Inquirers{holders, listingForTheJoiner{documentsJoiner}},
	)

	leadingWord := spread.leadingWordFinder.FindFor(ctx, query, documentInquiry)
	if lead, found := leadingWord.Get(); found {
		roles := wordroles.Around(lead, query)
		documentInquiry.ExpectInAPartition(lead.Word, lead.AmountOfDocumentsInAPartition)
		documentInquiry.WhichDocumentsHavingTheseAlsoHave(
			roles.LeadAndItsCompoundWords,
			roles.OtherWords,
		)
	} else {
		documentInquiry.WhichDocumentsHave(query.HashesOfWordsAndCompoundWords())
	}
	answered := documentInquiry.End()
	urlMetadataAnswers := urlMetadataLookup.End()

	documentsPerWord := documentsperword.From(query, answered, spread.partitions)
	joinedDocuments := documentsJoiner.JoinedDocuments()

	spread.documentAmountsCache.Remember(ctx, documentsPerWord.AmountInAPartitionPerQueryWord())
	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		answered, holders, documentsPerWord, leadingWord, joinedDocuments, urlMetadataAnswers,
		time.Since(startedAt),
	))

	return findingsFrom(
		query,
		answered,
		documentsPerWord,
		joinedDocuments,
		urlMetadataAnswers,
	)
}
