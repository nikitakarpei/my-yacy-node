package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func (s Spread) askForHeldDocuments(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord [][]peerdirectory.AskablePeer,
) ([]peerasks.HeldDocumentsAsk, []peerasks.AnsweredHeldDocumentsAsk) {
	asks := heldDocumentsAsksFor(query, chosenPeersPerQueryWord, s.peerItemsCeiling)
	firstRound, endFirstRound := contextOfTheFirstRound(ctx)
	defer endFirstRound()

	return asks, s.peerAsks.AskForHeldDocuments(firstRound, asks)
}

const amountOfRoundsOfPeerCalls = 2

func contextOfTheFirstRound(ctx context.Context) (context.Context, context.CancelFunc) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Until(deadline)/amountOfRoundsOfPeerCalls)
}
