// Package peerchoice chooses which peers each word of a query goes to. It
// spreads the words of one query over the DHT ring, favours the peers this
// deployment has found reliable, and reaches peers an earlier word or an
// earlier search did not.
package peerchoice

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const shareOfDistanceCountedForAFullyReliablePeer = 0.5

type PeerReliability interface {
	ReliabilityOf(
		ctx context.Context,
		peerAtAddress peeranswerhistory.PeerAtAddress,
	) float64
}

type PeerDirectory interface {
	MarkPeersChosen(ctx context.Context, peers []peerdirectory.AskablePeer)
}

type Choice struct {
	partitions      yacymodel.DHTRingPartitions
	peerReliability PeerReliability
	peerDirectory   PeerDirectory
	observer        PeerChoiceObserver
}

func New(
	partitions yacymodel.DHTRingPartitions,
	peerReliability PeerReliability,
	peerDirectory PeerDirectory,
	observer PeerChoiceObserver,
) Choice {
	return Choice{
		partitions:      partitions,
		peerReliability: peerReliability,
		peerDirectory:   peerDirectory,
		observer:        observer,
	}
}

func (c Choice) ChoosePeersPerQueryWord(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
	amountOfPeersHoldingOneWord int,
) [][]peerdirectory.AskablePeer {
	peersTheQueryMayAsk := c.peersOneQueryMayAsk(
		ctx, askablePeers, amountOfPeersHoldingOneWord,
	)
	chosenPeersPerQueryWord := make([][]peerdirectory.AskablePeer, 0, len(queryWords))
	for _, queryWord := range queryWords {
		chosenPeersPerQueryWord = append(
			chosenPeersPerQueryWord,
			peersTheQueryMayAsk.peersForQueryWord(
				ctx, queryWord, peersAcrossQueryWords(chosenPeersPerQueryWord),
			),
		)
	}
	c.peerDirectory.MarkPeersChosen(ctx, peersAcrossQueryWords(chosenPeersPerQueryWord))

	return chosenPeersPerQueryWord
}

func (c Choice) peersOneQueryMayAsk(
	ctx context.Context,
	askablePeers []peerdirectory.AskablePeer,
	amountOfPeersHoldingOneWord int,
) peersOneQueryMayAsk {
	shareOfDistanceCountedPerPeer := make(map[yacymodel.Hash]float64, len(askablePeers))
	for _, peer := range askablePeers {
		reliability := c.peerReliability.ReliabilityOf(
			ctx,
			peeranswerhistory.PeerAtAddress{Hash: peer.Hash, Address: peer.Address},
		)
		shareOfDistanceCountedPerPeer[peer.Hash] =
			1 - reliability*(1-shareOfDistanceCountedForAFullyReliablePeer)
	}

	return peersOneQueryMayAsk{
		partitions:                    c.partitions,
		observer:                      c.observer,
		askablePeers:                  askablePeers,
		shareOfDistanceCountedPerPeer: shareOfDistanceCountedPerPeer,
		amountOfPeersHoldingOneWord:   amountOfPeersHoldingOneWord,
	}
}

func peersAcrossQueryWords(
	peersPerQueryWord [][]peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	return slices.Concat(peersPerQueryWord...)
}
