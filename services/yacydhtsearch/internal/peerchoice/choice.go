// Package peerchoice chooses the peers the words of a query go to. It asks the
// same amount of peers for every word, picks the peers of each word in the
// order the words come in, and rests every peer it chooses so that the next
// search reaches further into the network.
package peerchoice

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerSelection interface {
	PeersForWord(
		ctx context.Context,
		word yacymodel.Hash,
		askablePeers []peerdirectory.AskablePeer,
		peersCeiling int,
	) []peerdirectory.AskablePeer
}

type PeerDirectory interface {
	MarkPeersChosen(ctx context.Context, peers []peerdirectory.AskablePeer)
}

type Choice struct {
	peerSelection PeerSelection
	peerDirectory PeerDirectory
}

func New(peerSelection PeerSelection, peerDirectory PeerDirectory) Choice {
	return Choice{peerSelection: peerSelection, peerDirectory: peerDirectory}
}

func (c Choice) ChoosePeersPerQueryWord(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
	amountOfPeersAskedPerWord int,
) [][]peerdirectory.AskablePeer {
	chosenPeersPerQueryWord := make([][]peerdirectory.AskablePeer, 0, len(queryWords))
	for _, queryWord := range queryWords {
		chosenPeers := c.peerSelection.PeersForWord(
			ctx,
			queryWord,
			askablePeers,
			amountOfPeersAskedPerWord,
		)
		c.peerDirectory.MarkPeersChosen(ctx, chosenPeers)
		chosenPeersPerQueryWord = append(chosenPeersPerQueryWord, chosenPeers)
	}

	return chosenPeersPerQueryWord
}
