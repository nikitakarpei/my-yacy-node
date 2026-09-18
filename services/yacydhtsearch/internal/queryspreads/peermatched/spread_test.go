package peermatched_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	itemsCeiling = 10
)

type peerNetwork struct {
	itemsPerPeer       map[string][]string
	peersCountingAWord map[string]struct{}
	silentPeers        map[string]struct{}
	asks               []peerasks.MatchedDocumentsAsk
}

func networkOf(itemsPerPeer map[string][]string) *peerNetwork {
	return &peerNetwork{
		itemsPerPeer:       itemsPerPeer,
		peersCountingAWord: map[string]struct{}{},
		silentPeers:        map[string]struct{}{},
	}
}

func (n *peerNetwork) AskForMatchedDocuments(
	_ context.Context,
	asks []peerasks.MatchedDocumentsAsk,
) []peerasks.AnsweredMatchedDocumentsAsk {
	n.asks = append(n.asks, asks...)

	answeredAsks := make([]peerasks.AnsweredMatchedDocumentsAsk, 0, len(asks))
	for _, ask := range asks {
		if _, silent := n.silentPeers[ask.Peer.Address]; silent {
			continue
		}
		answeredAsks = append(answeredAsks, peerasks.AnsweredMatchedDocumentsAsk{
			Ask:              ask,
			MatchedDocuments: n.matchedDocumentsOf(ask.Peer),
		})
	}

	return answeredAsks
}

func (n *peerNetwork) matchedDocumentsOf(
	peer peerdirectory.AskablePeer,
) []peerasks.MatchedDocument {
	_, countsAWord := n.peersCountingAWord[peer.Address]
	addresses := n.itemsPerPeer[peer.Address]
	matchedDocuments := make([]peerasks.MatchedDocument, 0, len(addresses))
	for _, address := range addresses {
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			continue
		}
		matchedDocument := peerasks.MatchedDocument{
			Metadata: yacymodel.URLMetadata{Hash: hash, Address: address},
		}
		if countsAWord {
			matchedDocument.Posting = yacymodel.Some(yacymodel.RWIPosting{Hits: 3})
		}
		matchedDocuments = append(matchedDocuments, matchedDocument)
	}

	return matchedDocuments
}

type everyAskablePeer struct{}

func (everyAskablePeer) ChosenPeersPerQueryWordFor(
	_ context.Context,
	queryWords []yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
) peerchoice.ChosenPeersPerQueryWord {
	peersPerQueryWord := make(peerchoice.ChosenPeersPerQueryWord, 0, len(queryWords))
	for _, queryWord := range queryWords {
		peersPerQueryWord = append(peersPerQueryWord, peerchoice.ChosenPeersOfQueryWord{
			QueryWord:   queryWord,
			ChosenPeers: chosenPeersInTheirOwnPartitions(askablePeers),
		})
	}

	return peersPerQueryWord
}

func chosenPeersInTheirOwnPartitions(
	askablePeers []peerdirectory.AskablePeer,
) []peerchoice.ChosenPeer {
	chosenPeers := make([]peerchoice.ChosenPeer, 0, len(askablePeers))
	for place, peer := range askablePeers {
		chosenPeers = append(
			chosenPeers,
			peerchoice.ChosenPeer{Peer: peer, Partition: uint(place)},
		)
	}

	return chosenPeers
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
) []queryanswers.FoundDocument {
	return spreadOf(network, observer).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom("berlin weather", ""),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	).FoundDocuments
}

func searchForTheQuery(network *peerNetwork, query string) []queryanswers.FoundDocument {
	return answersOfTheQuery(network, query).FoundDocuments
}

func addressesOf(foundDocuments []queryanswers.FoundDocument) []string {
	addresses := make([]string, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		addresses = append(addresses, foundDocument.Address)
	}

	return addresses
}

func answersOfTheQuery(network *peerNetwork, query string) queryanswers.AnsweredQuery {
	return spreadOf(network, &recordedSpreads{}).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom(query, ""),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	)
}

func spreadOf(
	network *peerNetwork,
	observer peermatched.PeerMatchedSpreadObserver,
) spreadChoosingEveryAskablePeer {
	return spreadChoosingEveryAskablePeer{spread: peermatched.New(network, itemsCeiling, observer)}
}

type spreadChoosingEveryAskablePeer struct {
	spread peermatched.Spread
}

func (s spreadChoosingEveryAskablePeer) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) queryanswers.AnsweredQuery {
	return s.spread.SpreadOverPeers(
		ctx,
		query,
		everyAskablePeer{}.ChosenPeersPerQueryWordFor(ctx, query.TermHashes(), askablePeers),
	)
}

func TestEveryPeerChosenForAnyQueryWordIsAskedOnce(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{})

	spreadOf(network, &recordedSpreads{}).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom("berlin weather", ""),
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

func TestADocumentSeveralPeersMatchedIsFoundOnce(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{
		"first":  {"https://a.example/", "https://shared.example/"},
		"second": {"https://shared.example/", "https://c.example/"},
	})

	foundDocuments := searchOf(network, &recordedSpreads{})

	want := []string{"https://a.example/", "https://shared.example/", "https://c.example/"}
	if got := addressesOf(foundDocuments); !slices.Equal(got, want) {
		t.Fatalf("the spread found %v, want %v", got, want)
	}
}

func TestADocumentKeepsTheCountOfAPeerThatCountedItsWord(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{
		"first":  {"https://shared.example/"},
		"second": {"https://shared.example/"},
	})
	network.peersCountingAWord["second"] = struct{}{}

	foundDocuments := searchForTheQuery(network, "berlin")

	if hits := foundDocuments[0].HitsPerQueryWord[yacymodel.WordHash("berlin")]; hits != 3 {
		t.Fatalf("the found document holds the hits %v, want the hits of the peer that counted "+
			"them", foundDocuments[0].HitsPerQueryWord)
	}
}

func TestTheAnswersCarryTheWordsOfTheQuery(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://a.example/"}})

	answers := answersOfTheQuery(network, "berlin weather")

	want := []yacymodel.Hash{yacymodel.WordHash("berlin"), yacymodel.WordHash("weather")}
	if !slices.Equal(answers.QueryWords, want) {
		t.Fatalf("the answers carry the query words %v, want %v", answers.QueryWords, want)
	}
}

func TestOnlyAQueryOfOneWordNamesTheWordAPeerCounted(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://a.example/"}})
	network.peersCountingAWord["first"] = struct{}{}

	ofOneWord := searchForTheQuery(network, "berlin")
	ofTwoWords := searchForTheQuery(network, "berlin weather")

	if hits := ofOneWord[0].HitsPerQueryWord[yacymodel.WordHash("berlin")]; hits != 3 {
		t.Fatalf("the found document of a one word query holds the hits %v, want the hits of "+
			"that word", ofOneWord[0].HitsPerQueryWord)
	}
	if ofTwoWords[0].CountedByAPeer() {
		t.Fatalf("the found document of a two word query reads %+v, want no count, because "+
			"the peer does not say which word it counted", ofTwoWords[0])
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

func TestEveryAskCarriesThePartitionOfTheChosenPeer(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{})

	searchOf(network, &recordedSpreads{})

	partitionsAsked := map[string]uint{}
	for _, ask := range network.asks {
		partitionsAsked[ask.Peer.Address] = ask.Partition
	}
	want := map[string]uint{"first": 0, "second": 1}
	if !maps.Equal(partitionsAsked, want) {
		t.Fatalf("the asks carry the partitions %v, want %v", partitionsAsked, want)
	}
}
