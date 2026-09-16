// Package peerchoice chooses which peers each word of a query goes to. For
// each place on the DHT ring a word is held, only the peers nearest to that
// place may hold the word, so it asks those peers and no others, and among them
// asks first the peers this deployment has found reliable. A later word of the
// query reaches peers an earlier word did not.
package peerchoice

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

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
		chosenPeers, ringFractionsOfTheTakenPeers := peersTheQueryMayAsk.peersForQueryWord(
			queryWord, peersAcrossQueryWords(chosenPeersPerQueryWord),
		)
		c.observer.PeersTakenFromTheRing(ctx, ringFractionsOfTheTakenPeers)
		chosenPeersPerQueryWord = append(chosenPeersPerQueryWord, chosenPeers)
	}
	c.peerDirectory.MarkPeersChosen(ctx, peersAcrossQueryWords(chosenPeersPerQueryWord))

	return chosenPeersPerQueryWord
}

func (c Choice) peersOneQueryMayAsk(
	ctx context.Context,
	askablePeers []peerdirectory.AskablePeer,
	amountOfPeersHoldingOneWord int,
) peersOneQueryMayAsk {
	reliabilityOfEachPeer := make(map[yacymodel.Hash]float64, len(askablePeers))
	for _, peer := range askablePeers {
		reliabilityOfEachPeer[peer.Hash] = c.peerReliability.ReliabilityOf(
			ctx,
			peeranswerhistory.PeerAtAddress{Hash: peer.Hash, Address: peer.Address},
		)
	}

	return peersOneQueryMayAsk{
		partitions:                  c.partitions,
		askablePeers:                askablePeers,
		reliabilityOfEachPeer:       reliabilityOfEachPeer,
		amountOfPeersHoldingOneWord: amountOfPeersHoldingOneWord,
	}
}

func peersAcrossQueryWords(
	peersPerQueryWord [][]peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	return slices.Concat(peersPerQueryWord...)
}
