// Package wordjoined finds documents that match the whole query when no single
// peer holds every query word, by joining what the peers of each word hold.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReplicaAsks interface {
	Start(ctx context.Context) wordpartitionasks.Run
}

type PeerAsks interface {
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) <-chan peerasks.URLMetadataAskOutcome
}

type QueryWordDocumentAmounts interface {
	DocumentAmountsOf(ctx context.Context, words []yacymodel.Hash) map[yacymodel.Hash]int
	Remember(ctx context.Context, documentAmounts map[yacymodel.Hash]int)
}

type URLMetadataAskCeilings interface {
	CeilingOf(ctx context.Context, address string) int
}

type Spread struct {
	replicaAsks                 ReplicaAsks
	peerAsks                    PeerAsks
	queryWordDocumentAmounts    QueryWordDocumentAmounts
	urlMetadataLookupCutoff     URLMetadataLookupCutoff
	partitionToSample           func(amountOfPartitions uint) uint
	urlMetadataAskCeilings      URLMetadataAskCeilings
	documentsToMatchCeiling     int
	partitions                  yacymodel.DHTRingPartitions
	amountOfPeersHoldingOneWord int
	observer                    WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the spread takes its asks, word amounts, lookup cutoff, partition to sample, ask ceilings, documents to match ceiling, ring and observer
func New(
	replicaAsks ReplicaAsks,
	peerAsks PeerAsks,
	queryWordDocumentAmounts QueryWordDocumentAmounts,
	urlMetadataLookupCutoff URLMetadataLookupCutoff,
	partitionToSample func(amountOfPartitions uint) uint,
	urlMetadataAskCeilings URLMetadataAskCeilings,
	documentsToMatchCeiling int,
	partitions yacymodel.DHTRingPartitions,
	amountOfPeersHoldingOneWord int,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		replicaAsks:                 replicaAsks,
		peerAsks:                    peerAsks,
		queryWordDocumentAmounts:    queryWordDocumentAmounts,
		urlMetadataLookupCutoff:     urlMetadataLookupCutoff,
		partitionToSample:           partitionToSample,
		urlMetadataAskCeilings:      urlMetadataAskCeilings,
		documentsToMatchCeiling:     documentsToMatchCeiling,
		partitions:                  partitions,
		amountOfPeersHoldingOneWord: amountOfPeersHoldingOneWord,
		observer:                    observer,
	}
}

func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	startedAt := time.Now()

	discoveryRound := spread.askToDiscover(ctx, query, chosenPeersPerQueryWord)
	spread.queryWordDocumentAmounts.Remember(
		ctx,
		discoveryRound.amountOfDocumentsHeldPerQueryWord(),
	)
	joinedDocuments := discoveryRound.joinedDocuments()
	urlMetadataLookupRound := spread.askForURLMetadata(ctx, discoveryRound, joinedDocuments)

	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		discoveryRound,
		joinedDocuments,
		urlMetadataLookupRound,
		time.Since(startedAt),
	))

	return answeredQueryFrom(
		query, discoveryRound, joinedDocuments, urlMetadataLookupRound,
	)
}

func (spread Spread) askToDiscover(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) discoveryRound {
	sampledPartition := spread.partitionToSample(uint(spread.partitions))
	rememberedDocumentAmounts := spread.queryWordDocumentAmounts.DocumentAmountsOf(
		ctx,
		query.WordHashes(),
	)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheDiscovery)
	defer endRound()

	return discoveryOver(
		startAskRun(roundContext, spread.replicaAsks),
		discoveryAsksFor(query, chosenPeersPerQueryWord),
		query,
		rememberedDocumentAmounts,
		spread.partitions,
		spread.documentsToMatchCeiling,
	).askTheQueryWords(sampledPartition)
}

const (
	roundsLeftAtTheDiscovery         = 2
	roundsLeftAtTheURLMetadataLookup = 1
)

func contextOfRound(ctx context.Context, roundsLeft int) (context.Context, context.CancelFunc) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Until(deadline)/time.Duration(roundsLeft))
}

func (spread Spread) askForURLMetadata(
	ctx context.Context,
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
) urlMetadataLookupRound {
	documentsWithoutMetadata := documentsWithoutMetadataAmong(
		joinedDocuments, discoveryRound.settledAsks.answers(),
	)
	asks := urlMetadataAsksFor(
		ctx,
		discoveryRound.holdersPerDocument.mostHeldFirst(documentsWithoutMetadata),
		discoveryRound.settledAsks.answers(),
		spread.urlMetadataAskCeilings,
		spread.amountOfPeersHoldingOneWord,
	)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheURLMetadataLookup)
	defer endRound()
	lookupContext, endLookup := context.WithCancel(roundContext)
	lookupInFlight := urlMetadataLookupInFlightOf(asks)
	endedLookup := lookupInFlight.settleUntilEnded(
		spread.peerAsks.AskForURLMetadata(lookupContext, asks),
		spread.urlMetadataLookupCutoff,
	)
	endLookup()

	return urlMetadataLookupRound{
		documentsWithoutMetadata: documentsWithoutMetadata,
		asks:                     asks,
		endedURLMetadataLookup:   endedLookup,
	}
}
