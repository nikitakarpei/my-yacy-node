package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

const ListsOnlyTheCrossCheckedDocuments peerjudgements.Question = "lists only the cross-checked documents"

func peersAtTheirVersionFrom(
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) []peerjudgements.PeerAtVersion {
	peers := chosenPeersPerQueryWord.PeersAcrossQueryWords()
	peersAtTheirVersion := make([]peerjudgements.PeerAtVersion, 0, len(peers))
	for _, peer := range peers {
		peersAtTheirVersion = append(
			peersAtTheirVersion,
			peerjudgements.PeerAtVersion{Peer: peer.Hash, Version: peer.Version},
		)
	}

	return peersAtTheirVersion
}

func chosenPeersLeastUsefulForTheCrossCheckFirst(
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	peerStandings peerjudgements.PeerStandings,
) peerchoice.ChosenPeersPerQueryWord {
	chosenPeersInOrder := make(peerchoice.ChosenPeersPerQueryWord, 0, len(chosenPeersPerQueryWord))
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		chosenPeers := slices.Clone(chosenPeersOfQueryWord.ChosenPeers)
		slices.SortStableFunc(chosenPeers, func(first, second peerchoice.ChosenPeer) int {
			return cmp.Compare(
				crossCheckUsefulnessOf(peerStandings.StandingOf(first.Peer.Hash)),
				crossCheckUsefulnessOf(peerStandings.StandingOf(second.Peer.Hash)),
			)
		})
		chosenPeersInOrder = append(chosenPeersInOrder, peerchoice.ChosenPeersOfQueryWord{
			QueryWord:   chosenPeersOfQueryWord.QueryWord,
			ChosenPeers: chosenPeers,
		})
	}

	return chosenPeersInOrder
}

type crossCheckUsefulness int

const (
	uselessForTheCrossCheck crossCheckUsefulness = iota
	unprovenForTheCrossCheck
	provenForTheCrossCheck
)

func crossCheckUsefulnessOf(standing peerjudgements.Standing) crossCheckUsefulness {
	switch standing {
	case peerjudgements.Ignoring:
		return uselessForTheCrossCheck
	case peerjudgements.Honoring:
		return provenForTheCrossCheck
	default:
		return unprovenForTheCrossCheck
	}
}

func peersUsefulForTheCrossCheckAmong(
	peers []peerdirectory.AskablePeer,
	peerStandings peerjudgements.PeerStandings,
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(peers))
	for _, peer := range peers {
		if crossCheckUsefulnessOf(peerStandings.StandingOf(peer.Hash)) == uselessForTheCrossCheck {
			continue
		}
		keptPeers = append(keptPeers, peer)
	}

	return keptPeers
}

func peersMostUsefulForTheCrossCheckFirst(
	peers []peerdirectory.AskablePeer,
	peerStandings peerjudgements.PeerStandings,
) []peerdirectory.AskablePeer {
	peersInOrder := slices.Clone(peers)
	slices.SortStableFunc(peersInOrder, func(first, second peerdirectory.AskablePeer) int {
		return cmp.Compare(
			crossCheckUsefulnessOf(peerStandings.StandingOf(second.Hash)),
			crossCheckUsefulnessOf(peerStandings.StandingOf(first.Hash)),
		)
	})

	return peersInOrder
}

func crossCheckJudgementsIn(round crossCheckedDocumentsRound) []peerjudgements.JudgedPeer {
	judgedPeers := make([]peerjudgements.JudgedPeer, 0, len(round.answeredAsks))
	for _, answeredAsk := range round.answeredAsks {
		judgedPeers = append(judgedPeers, peerjudgements.JudgedPeerFrom(
			answeredAsk.Ask.Peer.Hash,
			answeredAsk.Ask.Peer.Version,
			crossCheckJudgementOf(answeredAsk),
		))
	}

	return judgedPeers
}

func crossCheckJudgementOf(
	answeredAsk peerasks.AnsweredSearchDocumentsAsk,
) peerjudgements.Judgement {
	if len(answeredAsk.Abstract) == 0 {
		return peerjudgements.NoEvidence
	}
	if answeredAsk.IgnoredTheDocumentsToMatch() {
		return peerjudgements.Ignored
	}

	return peerjudgements.Honored
}
