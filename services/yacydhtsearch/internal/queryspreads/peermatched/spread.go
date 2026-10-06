// Package peermatched collects the documents each peer matched for a query of
// one word. Each time a settled ask adds documents it hands its findings so far
// to the growth.
package peermatched

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

type ReplicaAsks interface {
	Start(ctx context.Context) wordpartitionasks.Run
}

type Spread struct {
	replicaAsks ReplicaAsks
	observer    PeerMatchedSpreadObserver
}

func New(replicaAsks ReplicaAsks, observer PeerMatchedSpreadObserver) Spread {
	return Spread{replicaAsks: replicaAsks, observer: observer}
}

func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	growth queryfindings.Growth,
) queryfindings.Findings {
	startedAt := time.Now()

	asks := wordPartitionAsksFor(query, chosenPeersPerQueryWord)
	run := spread.replicaAsks.Start(ctx)
	run.Asks <- asks
	close(run.Asks)
	settledAsks := noSettledAsksYet(query, growth)
	settledAsks.keepAsTheySettle(run.SettledAsks)

	spread.observer.PeerMatchedSpreadPerformed(
		ctx,
		performedPeerMatchedSpreadFrom(
			query.WordHashes(),
			settledAsks.amountOfPeersThatMatchedNothing(),
			time.Since(startedAt),
		),
	)

	return settledAsks.findings()
}
