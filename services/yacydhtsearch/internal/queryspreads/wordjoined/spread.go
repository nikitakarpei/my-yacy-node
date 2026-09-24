// Package wordjoined finds documents that match the whole query when no single
// peer holds every query word, by joining what the peers of each word hold.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReplicaAsks interface {
	Start(ctx context.Context) replicaasks.Run
}

type PeerAsks interface {
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) <-chan peerasks.URLMetadataAskOutcome
}

type Spread struct {
	replicaAsks                    ReplicaAsks
	peerAsks                       PeerAsks
	partitionToSample              func(amountOfPartitions uint) uint
	urlMetadataAskDocumentsCeiling int
	documentsToMatchCeiling        int
	peerItemsCeiling               int
	partitions                     yacymodel.DHTRingPartitions
	amountOfPeersHoldingOneWord    int
	observer                       WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the spread takes its asks, partition to sample, ceilings, ring and observer
func New(
	replicaAsks ReplicaAsks,
	peerAsks PeerAsks,
	partitionToSample func(amountOfPartitions uint) uint,
	urlMetadataAskDocumentsCeiling int,
	documentsToMatchCeiling int,
	peerItemsCeiling int,
	partitions yacymodel.DHTRingPartitions,
	amountOfPeersHoldingOneWord int,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		replicaAsks:                    replicaAsks,
		peerAsks:                       peerAsks,
		partitionToSample:              partitionToSample,
		urlMetadataAskDocumentsCeiling: urlMetadataAskDocumentsCeiling,
		documentsToMatchCeiling:        documentsToMatchCeiling,
		peerItemsCeiling:               peerItemsCeiling,
		partitions:                     partitions,
		amountOfPeersHoldingOneWord:    amountOfPeersHoldingOneWord,
		observer:                       observer,
	}
}

func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	startedAt := time.Now()

	discoveryRound := spread.askToDiscover(ctx, query, chosenPeersPerQueryWord)
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
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheDiscovery)
	defer endRound()

	return discoveryOver(
		startAskRun(roundContext, spread.replicaAsks),
		discoveryAsksFor(query, chosenPeersPerQueryWord, spread.peerItemsCeiling),
		query,
		spread.partitions,
		spread.documentsToMatchCeiling,
	).askFromTheSampleIn(sampledPartition)
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
		joinedDocuments, discoveryRound.answeredAsks,
	)
	asks := urlMetadataAsksFor(
		discoveryRound.holdersPerDocument.mostHeldFirst(documentsWithoutMetadata),
		discoveryRound.answeredAsks,
		spread.urlMetadataAskDocumentsCeiling,
		spread.amountOfPeersHoldingOneWord,
	)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheURLMetadataLookup)
	defer endRound()
	lookupContext, endLookup := context.WithCancel(roundContext)
	answeredAsks, end := answeredAsksUntilCoverageFrom(
		spread.peerAsks.AskForURLMetadata(lookupContext, asks),
		lookedUpDocumentsAcross(asks),
	)
	endLookup()

	return urlMetadataLookupRound{
		documentsWithoutMetadata: documentsWithoutMetadata,
		asks:                     asks,
		answeredAsks:             answeredAsks,
		end:                      end,
	}
}
