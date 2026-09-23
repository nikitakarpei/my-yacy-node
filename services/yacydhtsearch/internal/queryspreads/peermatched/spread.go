// Package peermatched collects the documents each peer matched for a query of
// one word.
package peermatched

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type PeerAsks interface {
	AskForSearchDocuments(
		ctx context.Context,
		asks []peerasks.SearchDocumentsAsk,
	) peerasks.AsksPut[peerasks.SearchDocumentsAsk, peerasks.AnsweredSearchDocumentsAsk]
}

type Spread struct {
	peerAsks         PeerAsks
	peerItemsCeiling int
	observer         PeerMatchedSpreadObserver
}

func New(
	peerAsks PeerAsks,
	peerItemsCeiling int,
	observer PeerMatchedSpreadObserver,
) Spread {
	return Spread{
		peerAsks:         peerAsks,
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
	answeredAsks := spread.peerAsks.AskForSearchDocuments(ctx, asks).AnsweredAsks

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
