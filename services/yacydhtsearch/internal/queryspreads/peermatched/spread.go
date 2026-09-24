// Package peermatched collects the documents each peer matched for a query of
// one word.
package peermatched

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type ReplicaAsks interface {
	Start(ctx context.Context) replicaasks.Run
}

type Spread struct {
	replicaAsks      ReplicaAsks
	peerItemsCeiling int
	observer         PeerMatchedSpreadObserver
}

func New(
	replicaAsks ReplicaAsks,
	peerItemsCeiling int,
	observer PeerMatchedSpreadObserver,
) Spread {
	return Spread{
		replicaAsks:      replicaAsks,
		peerItemsCeiling: peerItemsCeiling,
		observer:         observer,
	}
}

func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	startedAt := time.Now()

	asks := searchDocumentsAsksFor(query, chosenPeersPerQueryWord, spread.peerItemsCeiling)
	run := spread.replicaAsks.Start(ctx)
	run.Asks <- asks
	close(run.Asks)
	answeredAsks := answeredAsksFrom(run.SettledWordPartitions)

	spread.observer.PeerMatchedSpreadPerformed(
		ctx,
		performedPeerMatchedSpreadFrom(
			query.WordHashes(),
			answeredAsks,
			time.Since(startedAt),
		),
	)

	return answeredQueryFrom(answeredAsks, query)
}
