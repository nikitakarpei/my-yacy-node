package peerchoice_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type nearestPeers struct {
	peers             []peerdirectory.AskablePeer
	peersCeilingAsked []int
}

func (n *nearestPeers) PeersForWord(
	_ context.Context,
	_ yacymodel.Hash,
	_ []peerdirectory.AskablePeer,
	peersCeiling int,
) []peerdirectory.AskablePeer {
	n.peersCeilingAsked = append(n.peersCeilingAsked, peersCeiling)

	return n.peers
}

type restedPeers struct {
	marked [][]peerdirectory.AskablePeer
}

func (r *restedPeers) MarkPeersChosen(_ context.Context, peers []peerdirectory.AskablePeer) {
	r.marked = append(r.marked, peers)
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash(address), Address: address}
}

func words(t *testing.T, spelled ...string) []yacymodel.Hash {
	t.Helper()

	hashes := make([]yacymodel.Hash, 0, len(spelled))
	for _, word := range spelled {
		hashes = append(hashes, yacymodel.WordHash(word))
	}

	return hashes
}

func TestEveryPeerChosenForAQueryWordRestsBeforeTheNextSearch(t *testing.T) {
	t.Parallel()

	selection := &nearestPeers{
		peers: []peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	}
	directory := &restedPeers{}

	peersPerQueryWord := peerchoice.New(selection, directory).ChoosePeersPerQueryWord(
		t.Context(),
		words(t, "berlin"),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second"), peerAt("third")},
		24,
	)

	if len(peersPerQueryWord) != 1 || len(peersPerQueryWord[0]) != 2 {
		t.Fatalf(
			"ChoosePeersPerQueryWord = %v, want the peers the selection named for the one word",
			peersPerQueryWord,
		)
	}
	if len(directory.marked) != 1 || len(directory.marked[0]) != 2 {
		t.Fatalf("the directory rested %v, want the peers that were chosen", directory.marked)
	}
}

func TestTheCallsOfOneQueryAreSharedOverItsWords(t *testing.T) {
	t.Parallel()

	selection := &nearestPeers{peers: []peerdirectory.AskablePeer{peerAt("first")}}

	peerchoice.New(selection, &restedPeers{}).ChoosePeersPerQueryWord(
		t.Context(),
		words(t, "berlin", "weather", "today"),
		[]peerdirectory.AskablePeer{peerAt("first")},
		24,
	)

	want := []int{8, 8, 8}
	if len(selection.peersCeilingAsked) != len(want) {
		t.Fatalf(
			"the selection was asked %d times, want one for each query word",
			len(selection.peersCeilingAsked),
		)
	}
	for index, ceiling := range selection.peersCeilingAsked {
		if ceiling != want[index] {
			t.Errorf("query word %d was given %d peers, want %d", index, ceiling, want[index])
		}
	}
}

func TestEveryQueryWordKeepsOnePeerWhenTheWordsOutnumberTheCalls(t *testing.T) {
	t.Parallel()

	selection := &nearestPeers{peers: []peerdirectory.AskablePeer{peerAt("first")}}

	peerchoice.New(selection, &restedPeers{}).ChoosePeersPerQueryWord(
		t.Context(),
		words(t, "berlin", "weather", "today"),
		[]peerdirectory.AskablePeer{peerAt("first")},
		2,
	)

	for index, ceiling := range selection.peersCeilingAsked {
		if ceiling != 1 {
			t.Errorf("query word %d was given %d peers, want one", index, ceiling)
		}
	}
}

func TestNoPeerRestsWhenTheQueryHasNoWord(t *testing.T) {
	t.Parallel()

	directory := &restedPeers{}

	peersPerQueryWord := peerchoice.New(&nearestPeers{}, directory).ChoosePeersPerQueryWord(
		t.Context(),
		nil,
		nil,
		24,
	)

	if len(peersPerQueryWord) != 0 {
		t.Fatalf("ChoosePeersPerQueryWord = %v, want none", peersPerQueryWord)
	}
	if len(directory.marked) != 0 {
		t.Fatalf("the directory rested %v, want nothing", directory.marked)
	}
}
