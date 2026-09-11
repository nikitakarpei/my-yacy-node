package peermatched_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	itemsCeiling        = 10
	peersHoldingOneWord = 24
)

type peerNetwork struct {
	itemsPerPeer            map[string][]string
	countsAWordWithEachItem bool
	silentPeers             map[string]struct{}
	asks                    []peerasks.MatchedItemsAsk
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
			Ask:              ask,
			MatchedDocuments: n.matchedDocumentsAt(n.itemsPerPeer[ask.Peer.Address]),
		})
	}

	return answeredAsks
}

func (n *peerNetwork) matchedDocumentsAt(addresses []string) []peerasks.MatchedDocument {
	matchedDocuments := make([]peerasks.MatchedDocument, 0, len(addresses))
	for _, address := range addresses {
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			continue
		}
		matchedDocument := peerasks.MatchedDocument{
			Metadata: yacymodel.URLMetadata{Hash: hash, Address: address},
		}
		if n.countsAWordWithEachItem {
			matchedDocument.CountOfAWordTheAskNamed = peeranswers.WordCount{Hits: 3}
		}
		matchedDocuments = append(matchedDocuments, matchedDocument)
	}

	return matchedDocuments
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

type recordedSpreads struct {
	performed []peermatched.PerformedPeerMatchedSpread
}

func (r *recordedSpreads) PeerMatchedSpreadPerformed(
	_ context.Context,
	spread peermatched.PerformedPeerMatchedSpread,
) {
	r.performed = append(r.performed, spread)
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash(address), Address: address}
}

func searchOf(
	network *peerNetwork,
	observer peermatched.PeerMatchedSpreadObserver,
) [][]peeranswers.AnsweredItem {
	return spreadOf(network, peersHoldingOneWord, observer).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom("berlin weather"),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	).ItemsInTheOrderOfEachAnswer
}

func searchForTheQuery(network *peerNetwork, query string) [][]peeranswers.AnsweredItem {
	return spreadOf(network, peersHoldingOneWord, &recordedSpreads{}).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom(query),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	).ItemsInTheOrderOfEachAnswer
}

func spreadOf(
	network *peerNetwork,
	peersOfOneWord int,
	observer peermatched.PeerMatchedSpreadObserver,
) peermatched.Spread {
	return peermatched.New(network, everyAskablePeer{}, itemsCeiling, peersOfOneWord, observer)
}

func TestEveryPeerChosenForAnyQueryWordIsAskedOnce(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{})

	spreadOf(network, peersHoldingOneWord, &recordedSpreads{}).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom("berlin weather"),
		[]peerdirectory.AskablePeer{
			peerAt("first"), peerAt("second"), peerAt("third"), peerAt("fourth"),
		},
	)

	if len(network.asks) != 4 {
		t.Fatalf("%d asks were put, want one for each chosen peer", len(network.asks))
	}
}

func TestEveryChosenPeerIsAskedToMatchTheWholeQueryOnce(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{})

	searchOf(network, &recordedSpreads{})

	if len(network.asks) != 2 {
		t.Fatalf("%d asks were put, want one for each chosen peer", len(network.asks))
	}
	for _, ask := range network.asks {
		if len(ask.WordsToMatch) != 2 || ask.ItemsCeiling != itemsCeiling {
			t.Fatalf("ask = %+v, want both query words", ask)
		}
	}
}

func TestTheItemsOfEachAnswerStayApart(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{
		"first":  {"https://a.example/", "https://b.example/"},
		"second": {"https://c.example/"},
	})

	itemsInTheOrderOfEachAnswer := searchOf(network, &recordedSpreads{})

	if len(itemsInTheOrderOfEachAnswer) != 2 {
		t.Fatalf(
			"%d peers answered items, want the items of each peer apart",
			len(itemsInTheOrderOfEachAnswer),
		)
	}
	if len(itemsInTheOrderOfEachAnswer[0])+len(itemsInTheOrderOfEachAnswer[1]) != 3 {
		t.Fatalf("the peers answered %v, want three items in total", itemsInTheOrderOfEachAnswer)
	}
}

func TestEveryItemAPeerMatchedMatchedEveryQueryWord(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://a.example/"}})

	itemsInTheOrderOfEachAnswer := searchOf(network, &recordedSpreads{})

	queryWords := []yacymodel.Hash{
		yacymodel.WordHash("berlin"), yacymodel.WordHash("weather"),
	}
	for _, items := range itemsInTheOrderOfEachAnswer {
		for _, item := range items {
			for _, queryWord := range queryWords {
				if _, matched := item.MatchedWords[queryWord]; !matched {
					t.Fatalf("the item of %v matched %v, want every query word",
						item.Metadata.Address, item.MatchedWords)
				}
			}
		}
	}
}

func TestOnlyAQueryOfOneWordNamesTheWordAPeerCounted(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://a.example/"}})
	network.countsAWordWithEachItem = true

	ofOneWord := searchForTheQuery(network, "berlin")
	ofTwoWords := searchForTheQuery(network, "berlin weather")

	if counted := ofOneWord[0][0].MatchedWords[yacymodel.WordHash("berlin")]; counted.Hits != 3 {
		t.Fatalf("the item of a one word query carries %+v, want the count under that word",
			ofOneWord[0][0].MatchedWords)
	}
	if ofTwoWords[0][0].CountedByAPeer() {
		t.Fatalf("the item of a two word query carries %+v, want no count, because the peer "+
			"does not say which word it counted", ofTwoWords[0][0].MatchedWords)
	}
}

func TestTheSpreadReportsHowManyAskedPeersAnswered(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://a.example/"}})
	network.silentPeers["second"] = struct{}{}
	observer := &recordedSpreads{}

	searchOf(network, observer)

	if len(observer.performed) != 1 {
		t.Fatalf("the observer saw %d spreads, want one", len(observer.performed))
	}
	performed := observer.performed[0]
	if performed.AmountOfPeersAsked != 2 || performed.AmountOfPeersThatAnswered != 1 {
		t.Fatalf("the spread reported %+v, want two asked peers and one that answered", performed)
	}
	if performed.TimeSpent <= 0 {
		t.Fatalf("the spread reported %v spent, want the time it took", performed.TimeSpent)
	}
}
