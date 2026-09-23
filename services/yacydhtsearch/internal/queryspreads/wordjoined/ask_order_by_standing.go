package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

func peersClaimingNoVersionFrom(
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) []peerjudgements.PeerAtVersion {
	peers := chosenPeersPerQueryWord.PeersAcrossQueryWords()
	peersClaimingNoVersion := make([]peerjudgements.PeerAtVersion, 0, len(peers))
	for _, peer := range peers {
		peersClaimingNoVersion = append(
			peersClaimingNoVersion,
			peerjudgements.PeerAtVersion{Peer: peer.Hash},
		)
	}

	return peersClaimingNoVersion
}

func chosenPeersInMatchedAndHeldDocumentsAskOrder(
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	peerStandings peerjudgements.PeerStandings,
) peerchoice.ChosenPeersPerQueryWord {
	chosenPeersInOrder := make(peerchoice.ChosenPeersPerQueryWord, 0, len(chosenPeersPerQueryWord))
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		chosenPeers := slices.Clone(chosenPeersOfQueryWord.ChosenPeers)
		slices.SortStableFunc(chosenPeers, func(first, second peerchoice.ChosenPeer) int {
			return cmp.Compare(
				matchedAndHeldDocumentsAskRankOf(peerStandings.StandingOf(first.Peer.Hash)),
				matchedAndHeldDocumentsAskRankOf(peerStandings.StandingOf(second.Peer.Hash)),
			)
		})
		chosenPeersInOrder = append(chosenPeersInOrder, peerchoice.ChosenPeersOfQueryWord{
			QueryWord:   chosenPeersOfQueryWord.QueryWord,
			ChosenPeers: chosenPeers,
		})
	}

	return chosenPeersInOrder
}

const (
	rankAskedFirst = iota
	rankAskedBetween
	rankAskedLast
)

func matchedAndHeldDocumentsAskRankOf(standing peerjudgements.Standing) int {
	switch standing {
	case peerjudgements.Ignoring:
		return rankAskedFirst
	case peerjudgements.Honoring:
		return rankAskedLast
	default:
		return rankAskedBetween
	}
}

func peersInCrossCheckedDocumentsAskOrder(
	peers []peerdirectory.AskablePeer,
	peerStandings peerjudgements.PeerStandings,
) []peerdirectory.AskablePeer {
	peersInOrder := slices.Clone(peers)
	slices.SortStableFunc(peersInOrder, func(first, second peerdirectory.AskablePeer) int {
		return cmp.Compare(
			crossCheckedDocumentsAskRankOf(peerStandings.StandingOf(first.Hash)),
			crossCheckedDocumentsAskRankOf(peerStandings.StandingOf(second.Hash)),
		)
	})

	return peersInOrder
}

func crossCheckedDocumentsAskRankOf(standing peerjudgements.Standing) int {
	if standing == peerjudgements.Honoring {
		return rankAskedFirst
	}

	return rankAskedLast
}
