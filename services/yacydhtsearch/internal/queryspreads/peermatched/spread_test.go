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
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	itemsCeiling = 10
)

type peerNetwork struct {
	itemsPerPeer       map[string][]string
	peersCountingAWord map[string]struct{}
	asks               []peerasks.SearchDocumentsAsk
}

func networkOf(itemsPerPeer map[string][]string) *peerNetwork {
	return &peerNetwork{
		itemsPerPeer:       itemsPerPeer,
		peersCountingAWord: map[string]struct{}{},
	}
}

func (network *peerNetwork) Start(_ context.Context) replicaasks.Run {
	asks := make(chan []peerasks.SearchDocumentsAsk)
	settledWordPartitions := make(chan replicaasks.SettledWordPartition)
	go network.answerEachAsk(asks, settledWordPartitions)

	return replicaasks.Run{Asks: asks, SettledWordPartitions: settledWordPartitions}
}

func (network *peerNetwork) answerEachAsk(
	asks <-chan []peerasks.SearchDocumentsAsk,
	settledWordPartitions chan<- replicaasks.SettledWordPartition,
) {
	defer close(settledWordPartitions)
	for addedAsks := range asks {
		network.asks = append(network.asks, addedAsks...)
		for _, ask := range slices.Backward(addedAsks) {
			settledWordPartitions <- network.settledWordPartitionOf(ask)
		}
	}
}

func (network *peerNetwork) settledWordPartitionOf(
	ask peerasks.SearchDocumentsAsk,
) replicaasks.SettledWordPartition {
	return replicaasks.SettledWordPartition{AskOutcomes: peerasks.SearchDocumentsAskOutcomes{{
		Ask: ask,
		Put: true,
		Answer: yacymodel.Some(peerasks.AnsweredSearchDocumentsAsk{
			Ask:              ask,
			MatchedDocuments: network.matchedDocumentsOf(ask.Peer),
		}),
	}}}
}

func (network *peerNetwork) matchedDocumentsOf(
	peer peerdirectory.AskablePeer,
) []peerasks.MatchedDocument {
	_, countsAWord := network.peersCountingAWord[peer.Address]
	addresses := network.itemsPerPeer[peer.Address]
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
			matchedDocument.Posting = yacymodel.Some(yacymodel.RWIPosting{
				Hits:          3,
				LocalLinks:    12,
				ExternalLinks: 7,
			})
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
		searchquery.QueryFrom("berlin", ""),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	).FoundDocuments
}

func factsOfTheFirstDocumentFoundFor(
	network *peerNetwork, query string,
) queryanswers.DocumentFacts {
	answers := answersOfTheQuery(network, query)

	return answers.FoundDocuments[0].Facts
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
		everyAskablePeer{}.ChosenPeersPerQueryWordFor(ctx, query.WordHashes(), askablePeers),
	)
}

func TestEveryChosenPeerIsAskedToMatchTheQueryWordOnce(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{})

	searchOf(network, &recordedSpreads{})

	if len(network.asks) != 2 {
		t.Fatalf("%d asks were put, want one for each chosen peer", len(network.asks))
	}
	for _, ask := range network.asks {
		if ask.Word != yacymodel.WordHash("berlin") || ask.ItemsCeiling != itemsCeiling {
			t.Fatalf("ask = %+v, want the query word", ask)
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

	want := []string{"https://a.example/", "https://c.example/", "https://shared.example/"}
	if got := slices.Sorted(slices.Values(addressesOf(foundDocuments))); !slices.Equal(got, want) {
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

	facts := factsOfTheFirstDocumentFoundFor(network, "berlin")

	if hits := facts.HitsPerQueryWord[yacymodel.WordHash("berlin")]; hits != 3 {
		t.Fatalf("the found document holds the hits %v, want the hits of the peer that counted "+
			"them", facts.HitsPerQueryWord)
	}
}

func TestTheAnswersCarryTheWordsOfTheQuery(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://a.example/"}})

	answers := answersOfTheQuery(network, "berlin")

	want := []yacymodel.Hash{yacymodel.WordHash("berlin")}
	if !slices.Equal(answers.QueryWords, want) {
		t.Fatalf("the answers carry the query words %v, want %v", answers.QueryWords, want)
	}
}

func TestTheSpreadReportsTheTimeItTook(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://a.example/"}})
	observer := &recordedSpreads{}

	searchOf(network, observer)

	if len(observer.performed) != 1 {
		t.Fatalf("the observer saw %d spreads, want one", len(observer.performed))
	}
	performed := observer.performed[0]
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

func TestADocumentKeepsTheAmountOfLinksOfAPeerThatCountedItsWord(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://shared.example/"}})
	network.peersCountingAWord["first"] = struct{}{}

	facts := factsOfTheFirstDocumentFoundFor(network, "berlin")

	amountOfLinks, reported := facts.AmountOfLinks.Get()
	if !reported || amountOfLinks != 19 {
		t.Fatalf(
			"the found document holds %d links reported %t, want the 19 links the peer counted",
			amountOfLinks,
			reported,
		)
	}
}

func TestADocumentNoPeerCountedHoldsNoAmountOfLinks(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string][]string{"first": {"https://shared.example/"}})

	facts := factsOfTheFirstDocumentFoundFor(network, "berlin")

	if facts.AmountOfLinks.Present() {
		t.Fatal("the found document holds an amount of links, want none where no peer reported one")
	}
}
