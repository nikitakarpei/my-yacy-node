// Package wordjoined finds documents that match the whole query when no single
// peer holds every query word, by joining what the peers of each word hold. Each
// time it finds more documents it hands its findings so far to the growth, and in
// the end it returns all it found.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentjoin"
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
	growth queryfindings.Growth,
) queryfindings.Findings {
	startedAt := time.Now()
	holders := documentholders.NoneYet()
	measurement := documentamounts.NoneMeasuredYet(query, spread.partitions)
	found := noDocumentsFoundYet(query, measurement, growth)
	urlMetadataLookup := spread.urlMetadataAsker.Begin(ctx, found)
	documentsJoiner := documentjoin.JoinerOf(query, documentjoin.JoinObservers{
		found,
		urlMetadataLookupOfJoinedDocuments{holders, urlMetadataLookup},
	})
	documentInquiry := spread.documentsAsker.Begin(
		ctx, query, chosenPeersPerQueryWord,
		documentasks.Inquirers{holders, measurement, found, joinOfListedDocuments{documentsJoiner}},
	)

	leadingWord := spread.leadingWordFinder.FindFor(ctx, query)
	if lead, led := leadingWord.Get(); led {
		roles := wordroles.Around(lead, query)
		documentInquiry.ExpectInAPartition(lead.Word, lead.AmountOfDocumentsInAPartition)
		documentInquiry.WhichDocumentsHavingTheseAlsoHave(
			roles.LeadAndItsCompoundWords,
			roles.OtherWords,
		)
	} else {
		documentInquiry.WhichDocumentsHave(query.HashesOfWordsAndCompoundWords())
	}
	documentInquiry.End()
	urlMetadataLookup.End()
	joinedDocuments := documentsJoiner.JoinedDocuments()

	spread.documentAmountsCache.Remember(ctx, measurement.InAPartitionPerQueryWord())
	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		query, holders, measurement, leadingWord, joinedDocuments, time.Since(startedAt),
	))

	return found.findings()
}
