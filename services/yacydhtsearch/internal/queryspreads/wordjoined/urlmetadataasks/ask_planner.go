package urlmetadataasks

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askPlanner struct {
	ceilings        Ceilings
	asksPerDocument int
}

func (planner askPlanner) asksFor(
	ctx context.Context,
	holders documentholders.Holders,
) []peerasks.URLMetadataAsk {
	plan := askPlanFrom(
		planner.ceilingOfEachPeerAmong(ctx, holders.Peers()),
		holders.AmountOfDocumentsOfEachPeer(),
		planner.asksPerDocument,
	)
	for _, document := range holders.LeastHeldFirst() {
		plan.assign(document, holders.PeersHolding(document))
	}

	return plan.asks()
}

func (planner askPlanner) ceilingOfEachPeerAmong(
	ctx context.Context,
	peers []peerdirectory.AskablePeer,
) map[yacymodel.Hash]int {
	ceilingOfEachPeer := make(map[yacymodel.Hash]int, len(peers))
	for _, peer := range peers {
		ceilingOfEachPeer[peer.Hash] = planner.ceilings.CeilingOf(ctx, peer.Address)
	}

	return ceilingOfEachPeer
}
