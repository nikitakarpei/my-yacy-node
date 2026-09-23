package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

func peersAtNoVersionAmong(
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) []peerjudgements.PeerAtVersion {
	peers := chosenPeersPerQueryWord.PeersAcrossQueryWords()
	peersAtNoVersion := make([]peerjudgements.PeerAtVersion, 0, len(peers))
	for _, peer := range peers {
		peersAtNoVersion = append(peersAtNoVersion, peerjudgements.PeerAtVersion{Peer: peer.Hash})
	}

	return peersAtNoVersion
}

func chosenPeersInFirstRoundOrder(
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	peerStandings peerjudgements.PeerStandings,
) peerchoice.ChosenPeersPerQueryWord {
	chosenPeersInOrder := make(peerchoice.ChosenPeersPerQueryWord, 0, len(chosenPeersPerQueryWord))
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		chosenPeers := slices.Clone(chosenPeersOfQueryWord.ChosenPeers)
		slices.SortStableFunc(chosenPeers, func(first, second peerchoice.ChosenPeer) int {
			return cmp.Compare(
				firstRoundRankOf(peerStandings.StandingOf(first.Peer.Hash)),
				firstRoundRankOf(peerStandings.StandingOf(second.Peer.Hash)),
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

func firstRoundRankOf(standing peerjudgements.Standing) int {
	switch standing {
	case peerjudgements.Ignoring:
		return rankAskedFirst
	case peerjudgements.Honoring:
		return rankAskedLast
	default:
		return rankAskedBetween
	}
}

func peersInSecondRoundOrder(
	peers []peerdirectory.AskablePeer,
	peerStandings peerjudgements.PeerStandings,
) []peerdirectory.AskablePeer {
	peersInOrder := slices.Clone(peers)
	slices.SortStableFunc(peersInOrder, func(first, second peerdirectory.AskablePeer) int {
		return cmp.Compare(
			secondRoundRankOf(peerStandings.StandingOf(first.Hash)),
			secondRoundRankOf(peerStandings.StandingOf(second.Hash)),
		)
	})

	return peersInOrder
}

func secondRoundRankOf(standing peerjudgements.Standing) int {
	if standing == peerjudgements.Honoring {
		return rankAskedFirst
	}

	return rankAskedBetween
}
