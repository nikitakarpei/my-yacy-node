package peermatched_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	itemsCeiling     = 10
	peerCallsCeiling = 24
)

type peerNetwork struct {
	itemsPerPeer map[string][]string
	silentPeers  map[string]struct{}
	asks         []peerasks.MatchedItemsAsk
}

func networkOf(itemsPerPeer map[string][]string) *peerNetwork {
	return &peerNetwork{itemsPerPeer: itemsPerPeer, silentPeers: map[string]struct{}{}}
}

func (n *peerNetwork) AskForMatchedItems(
	_ context.Context,
	asks []peerasks.MatchedItemsAsk,
) []peerasks.AnsweredMatchedItemsAsk {
	n.asks = append(n.asks, asks...)

	answeredAsks := make([]peerasks.AnsweredMatchedItemsAsk, 0, len(asks))
	for _, ask := range asks {
		if _, silent := n.silentPeers[ask.Peer.Address]; silent {
			continue
		}
		answeredAsks = append(answeredAsks, peerasks.AnsweredMatchedItemsAsk{
			Ask:   ask,
			Items: itemsAt(n.itemsPerPeer[ask.Peer.Address]),
		})
	}

	return answeredAsks
}

func itemsAt(addresses []string) []searchresult.Item {
	items := make([]searchresult.Item, 0, len(addresses))
	for _, address := range addresses {
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			continue
		}
		items = append(
			items,
			searchresult.ItemFrom(yacymodel.URLMetadata{Hash: hash, Address: address}),
		)
	}

	return items
}

type everyAskablePeer struct{}

func (everyAskablePeer) ChoosePeersPerQueryWord(
	_ context.Context,
	queryWords []yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
	_ int,
) [][]peerdirectory.AskablePeer {
	peersPerQueryWord := make([][]peerdirectory.AskablePeer, 0, len(queryWords))
	for range queryWords {
		peersPerQueryWord = append(peersPerQueryWord, askablePeers)
	}

	return peersPerQueryWord
}

type recordedSearches struct {
	performed []peermatched.PerformedPeerMatchedSearch
}

func (r *recordedSearches) PeerMatchedSearchPerformed(
	_ context.Context,
	search peermatched.PerformedPeerMatchedSearch,
) {
	r.performed = append(r.performed, search)
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash(address), Address: address}
}

func searchOf(
	network *peerNetwork,
	observer peermatched.PeerMatchedSearchObserver,
) [][]searchresult.Item {
	return spreadOf(network, peerCallsCeiling, observer).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom("berlin weather"),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	)
}

func spreadOf(
	network *peerNetwork,
	peerCalls int,
	observer peermatched.PeerMatchedSearchObserver,
) peermatched.Spread {
	return peermatched.New(network, everyAskablePeer{}, itemsCeiling, peerCalls, observer)
}

func TestNoMorePeersAreAskedThanTheCallsOneQueryMayPut(t *testing.T) {
	t.Parallel()

	const peerCalls = 2

	network := networkOf(map[string][]string{})

	spreadOf(network, peerCalls, &recordedSearches{}).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom("berlin weather"),
		[]peerdirectory.AskablePeer{
			peerAt("first"), peerAt("second"), peerAt("third"), peerAt("fourth"),
		},
	)

	if len(network.asks) != peerCalls {
		t.Fatalf("%d asks were put, want %d", len(network.asks), peerCalls)
	}
}

func TestEveryChosenPeerIsAskedToMatchTheWholeQueryOnce(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{})

	searchOf(network, &recordedSearches{})

	if len(network.asks) != 2 {
		t.Fatalf("%d asks were put, want one for each chosen peer", len(network.asks))
	}
	for _, ask := range network.asks {
		if len(ask.WordsToMatch) != 2 || ask.ItemsCeiling != itemsCeiling {
			t.Fatalf("ask = %+v, want both query words", ask)
		}
	}
}

func TestTheItemsOfEachPeerStayApart(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{
		"first":  {"https://a.example/", "https://b.example/"},
		"second": {"https://c.example/"},
	})

	itemsOfEachPeer := searchOf(network, &recordedSearches{})

	if len(itemsOfEachPeer) != 2 {
		t.Fatalf("%d peers answered items, want the items of each peer apart", len(itemsOfEachPeer))
	}
	if len(itemsOfEachPeer[0])+len(itemsOfEachPeer[1]) != 3 {
		t.Fatalf("the peers answered %v, want three items in total", itemsOfEachPeer)
	}
}

func TestTheSearchReportsHowManyAskedPeersAnswered(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://a.example/"}})
	network.silentPeers["second"] = struct{}{}
	observer := &recordedSearches{}

	searchOf(network, observer)

	if len(observer.performed) != 1 {
		t.Fatalf("the observer saw %d searches, want one", len(observer.performed))
	}
	performed := observer.performed[0]
	if performed.AmountOfAskedPeers != 2 || performed.AmountOfPeersThatAnswered != 1 {
		t.Fatalf("the search reported %+v, want two asked peers and one that answered", performed)
	}
	if performed.TimeSpent <= 0 {
		t.Fatalf("the search reported %v spent, want the time it took", performed.TimeSpent)
	}
}
