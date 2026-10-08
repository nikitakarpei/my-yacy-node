package urlmetadataasks

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askPlan struct {
	asksPerDocument      int
	spaceOfEachPeer      map[yacymodel.Hash]int
	plannedAskOfEachPeer map[yacymodel.Hash]peerasks.URLMetadataAsk
}

func askPlanFrom(ceilingOfEachPeer map[yacymodel.Hash]int, asksPerDocument int) *askPlan {
	return &askPlan{
		asksPerDocument:      asksPerDocument,
		spaceOfEachPeer:      ceilingOfEachPeer,
		plannedAskOfEachPeer: map[yacymodel.Hash]peerasks.URLMetadataAsk{},
	}
}

func (plan *askPlan) assign(document yacymodel.URLHash, holders []peerdirectory.AskablePeer) {
	for _, holder := range plan.holdersChosenAmong(holders) {
		plan.assignTo(holder, document)
	}
}

func (plan *askPlan) holdersChosenAmong(
	holders []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	holdersWithSpace := slices.DeleteFunc(
		holders,
		func(holder peerdirectory.AskablePeer) bool { return plan.spaceOfEachPeer[holder.Hash] == 0 },
	)
	slices.SortFunc(holdersWithSpace, plan.plannedThenMostSpaceFirst)

	return holdersWithSpace[:min(plan.asksPerDocument, len(holdersWithSpace))]
}

func (plan *askPlan) plannedThenMostSpaceFirst(first, second peerdirectory.AskablePeer) int {
	_, firstPlanned := plan.plannedAskOfEachPeer[first.Hash]
	_, secondPlanned := plan.plannedAskOfEachPeer[second.Hash]
	if firstPlanned != secondPlanned {
		if firstPlanned {
			return -1
		}

		return 1
	}

	return cmp.Or(
		cmp.Compare(plan.spaceOfEachPeer[second.Hash], plan.spaceOfEachPeer[first.Hash]),
		strings.Compare(first.Hash.String(), second.Hash.String()),
	)
}

func (plan *askPlan) assignTo(holder peerdirectory.AskablePeer, document yacymodel.URLHash) {
	plannedAsk := plan.plannedAskOfEachPeer[holder.Hash]
	plannedAsk.Peer = holder
	plannedAsk.Documents = append(plannedAsk.Documents, document)
	plan.plannedAskOfEachPeer[holder.Hash] = plannedAsk
	plan.spaceOfEachPeer[holder.Hash]--
}

func (plan *askPlan) asks() []peerasks.URLMetadataAsk {
	return slices.SortedFunc(
		maps.Values(plan.plannedAskOfEachPeer),
		func(first, second peerasks.URLMetadataAsk) int {
			return strings.Compare(first.Peer.Hash.String(), second.Peer.Hash.String())
		},
	)
}
