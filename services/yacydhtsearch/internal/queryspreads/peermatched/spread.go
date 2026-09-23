// Package peermatched collects the documents each peer matched for the whole
// query on its own.
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
	AskForMatchedDocuments(
		ctx context.Context,
		asks []peerasks.MatchedDocumentsAsk,
	) peerasks.AsksPut[peerasks.MatchedDocumentsAsk, peerasks.AnsweredMatchedDocumentsAsk]
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

	chosenPeers := chosenPeersPerQueryWord.ChosenPeersAcrossQueryWords()
	asks := matchedDocumentsAsksFor(query, chosenPeers, spread.peerItemsCeiling)
	answeredAsks := spread.peerAsks.AskForMatchedDocuments(ctx, asks).AnsweredAsks

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
