// Package peerchoice chooses which peers each word of a query goes to. For
// each place on the DHT ring a word is held, only the peers nearest to that
// place may hold the word, so it asks the nearest peers first, and among the
// peers that hold the word asks first the ones this deployment has found
// reliable. A later word of the query reaches peers an earlier word did not.
package peerchoice

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerReliability interface {
	ReliabilityOf(
		ctx context.Context,
		peerAtAddress probeanswerhistory.PeerAtAddress,
	) float64
}

type PeerDirectory interface {
	MarkPeersChosen(ctx context.Context, peers []peerdirectory.AskablePeer)
}

type Choice struct {
	partitions        yacymodel.DHTRingPartitions
	networkRedundancy int
	peerReliability   PeerReliability
	peerDirectory     PeerDirectory
	observer          PeerChoiceObserver
}

func New(
	partitions yacymodel.DHTRingPartitions,
	networkRedundancy int,
	peerReliability PeerReliability,
	peerDirectory PeerDirectory,
	observer PeerChoiceObserver,
) Choice {
	return Choice{
		partitions:        partitions,
		networkRedundancy: networkRedundancy,
		peerReliability:   peerReliability,
		peerDirectory:     peerDirectory,
		observer:          observer,
	}
}

func (c Choice) ChoosePeersPerQueryWord(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
) ChosenPeersPerQueryWord {
	peersTheQueryMayAsk := c.peersOneQueryMayAsk(ctx, askablePeers)
	chosenPeersPerQueryWord := make(ChosenPeersPerQueryWord, 0, len(queryWords))
	for _, queryWord := range queryWords {
		chosenPeers, ringFractionsOfTheTakenPeers := peersTheQueryMayAsk.peersForQueryWord(
			queryWord, chosenPeersPerQueryWord.PeersAcrossQueryWords(),
		)
		c.observer.PeersTakenFromTheRing(ctx, ringFractionsOfTheTakenPeers)
		chosenPeersPerQueryWord = append(chosenPeersPerQueryWord, ChosenPeersOfQueryWord{
			QueryWord:   queryWord,
			ChosenPeers: chosenPeers,
		})
	}
	c.peerDirectory.MarkPeersChosen(ctx, chosenPeersPerQueryWord.PeersAcrossQueryWords())

	return chosenPeersPerQueryWord
}

func (c Choice) peersOneQueryMayAsk(
	ctx context.Context,
	askablePeers []peerdirectory.AskablePeer,
) peersOneQueryMayAsk {
	reliabilityOfEachPeer := make(map[yacymodel.Hash]float64, len(askablePeers))
	for _, peer := range askablePeers {
		reliabilityOfEachPeer[peer.Hash] = c.peerReliability.ReliabilityOf(
			ctx,
			probeanswerhistory.PeerAtAddress{Hash: peer.Hash, Address: peer.Address},
		)
	}

	return peersOneQueryMayAsk{
		partitions:            c.partitions,
		networkRedundancy:     c.networkRedundancy,
		askablePeers:          askablePeers,
		reliabilityOfEachPeer: reliabilityOfEachPeer,
	}
}
