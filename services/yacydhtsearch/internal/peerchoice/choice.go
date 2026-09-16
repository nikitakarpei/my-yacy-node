// Package peerchoice chooses the peers the words of a query go to. It asks the
// same amount of peers for every word and picks the peers of one word by how
// near they sit to the postings of that word on the DHT ring, taking the peers
// from every partition of the ring in turn so that a ceiling below what the
// partitions hold still leaves every partition covered. A peer counts as nearer
// to a posting than it is by how reliable this deployment has found it, and a
// peer this query has already asked comes after every peer it has not, so the
// words of one query reach peers the earlier words did not. A share of every
// word's peers is drawn at random from the peers the ring did not pick, so a
// peer this deployment knows nothing about is asked as well. Every peer the
// query chooses then rests, so that the next search reaches further.
package peerchoice

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const shareOfDistanceKeptByAFullyReliablePeer = 0.5

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
	amountOfPeersAskedPerWord int,
) [][]peerdirectory.AskablePeer {
	oneQuery := c.choiceForOneQuery(ctx, askablePeers, amountOfPeersAskedPerWord)
	chosenPeersPerQueryWord := make([][]peerdirectory.AskablePeer, 0, len(queryWords))
	for _, queryWord := range queryWords {
		chosenPeersPerQueryWord = append(chosenPeersPerQueryWord, oneQuery.peersForWord(
			ctx, queryWord, peersAcrossQueryWords(chosenPeersPerQueryWord),
		))
	}
	c.peerDirectory.MarkPeersChosen(ctx, peersAcrossQueryWords(chosenPeersPerQueryWord))

	return chosenPeersPerQueryWord
}

func (c Choice) choiceForOneQuery(
	ctx context.Context,
	askablePeers []peerdirectory.AskablePeer,
	peersCeiling int,
) choiceForOneQuery {
	shareOfDistanceCountedPerPeer := make(
		map[peerdirectory.AskablePeer]float64, len(askablePeers),
	)
	for _, peer := range askablePeers {
		reliability := c.peerReliability.ReliabilityOf(
			ctx,
			peeranswerhistory.PeerAtAddress{Hash: peer.Hash, Address: peer.Address},
		)
		shareOfDistanceCountedPerPeer[peer] =
			1 - reliability*(1-shareOfDistanceKeptByAFullyReliablePeer)
	}

	return choiceForOneQuery{
		Choice:                        c,
		askablePeers:                  askablePeers,
		shareOfDistanceCountedPerPeer: shareOfDistanceCountedPerPeer,
		peersCeiling:                  peersCeiling,
	}
}

func peersAcrossQueryWords(
	peersPerQueryWord [][]peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	return slices.Concat(peersPerQueryWord...)
}
