// Package peerchoice chooses the peers the words of a query go to. It shares
// the peer calls one query may put over the words of that query, picks the
// peers of each word in the order the words come in, and rests every peer it
// chooses so that the next search reaches further into the network.
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
	peerCallsCeiling int,
) [][]peerdirectory.AskablePeer {
	peersCeiling := peersCeilingPerQueryWord(peerCallsCeiling, len(queryWords))

	peersPerQueryWord := make([][]peerdirectory.AskablePeer, 0, len(queryWords))
	for _, queryWord := range queryWords {
		chosenPeers := c.peerSelection.PeersForWord(ctx, queryWord, askablePeers, peersCeiling)
		c.peerDirectory.MarkPeersChosen(ctx, chosenPeers)
		peersPerQueryWord = append(peersPerQueryWord, chosenPeers)
	}

	return peersPerQueryWord
}

// peersCeilingPerQueryWord shares the peer calls one round of a query may put
// evenly over the words of that query, and leaves every word at least one peer.
func peersCeilingPerQueryWord(peerCallsCeiling, queryWords int) int {
	if queryWords == 0 {
		return 0
	}

	return max(1, peerCallsCeiling/queryWords)
}
