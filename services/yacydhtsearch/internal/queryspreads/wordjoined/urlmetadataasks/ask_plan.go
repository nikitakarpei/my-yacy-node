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
	asksPerDocument             int
	spaceOfEachPeer             map[yacymodel.Hash]int
	amountOfDocumentsOfEachPeer map[yacymodel.Hash]int
	plannedAskOfEachPeer        map[yacymodel.Hash]peerasks.URLMetadataAsk
}

func askPlanFrom(
	ceilingOfEachPeer map[yacymodel.Hash]int,
	amountOfDocumentsOfEachPeer map[yacymodel.Hash]int,
	asksPerDocument int,
) *askPlan {
	return &askPlan{
		asksPerDocument:             asksPerDocument,
		spaceOfEachPeer:             ceilingOfEachPeer,
		amountOfDocumentsOfEachPeer: amountOfDocumentsOfEachPeer,
		plannedAskOfEachPeer:        map[yacymodel.Hash]peerasks.URLMetadataAsk{},
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
		slices.Clone(holders),
		func(holder peerdirectory.AskablePeer) bool { return plan.spaceOfEachPeer[holder.Hash] == 0 },
	)
	slices.SortFunc(holdersWithSpace, plan.preferredFirst)

	return holdersWithSpace[:min(plan.asksPerDocument, len(holdersWithSpace))]
}

func (plan *askPlan) preferredFirst(first, second peerdirectory.AskablePeer) int {
	_, firstAssigned := plan.plannedAskOfEachPeer[first.Hash]
	_, secondAssigned := plan.plannedAskOfEachPeer[second.Hash]
	if firstAssigned && !secondAssigned {
		return -1
	}
	if secondAssigned && !firstAssigned {
		return 1
	}

	return cmp.Or(
		cmp.Compare(
			plan.amountOfDocumentsOfEachPeer[second.Hash],
			plan.amountOfDocumentsOfEachPeer[first.Hash],
		),
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
