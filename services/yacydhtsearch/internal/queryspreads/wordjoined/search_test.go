package wordjoined_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord                        = "berlin"
	secondWord                       = "weather"
	documentsToAskMetadataForCeiling = 10
	peerCallsCeiling                 = 24
)

type peerNetwork struct {
	documentsPerWordPerPeer map[string]map[string][]string
	heldDocumentsAsks       []peerasks.HeldDocumentsAsk
	urlMetadataAsks         []peerasks.URLMetadataAsk
	silentPeers             map[string]struct{}
}

func networkOf(documentsPerWordPerPeer map[string]map[string][]string) *peerNetwork {
	return &peerNetwork{
		documentsPerWordPerPeer: documentsPerWordPerPeer,
		silentPeers:             map[string]struct{}{},
	}
}

func (n *peerNetwork) AskForHeldDocuments(
	_ context.Context,
	asks []peerasks.HeldDocumentsAsk,
) []peerasks.AnsweredHeldDocumentsAsk {
	n.heldDocumentsAsks = append(n.heldDocumentsAsks, asks...)

	answeredAsks := make([]peerasks.AnsweredHeldDocumentsAsk, 0, len(asks))
	for _, ask := range asks {
		if _, silent := n.silentPeers[ask.Peer.Address]; silent {
			continue
		}
		answeredAsks = append(answeredAsks, peerasks.AnsweredHeldDocumentsAsk{
			Ask:       ask,
			Documents: n.documentsHeldBy(ask.Peer.Address, ask.Word),
		})
	}

	return answeredAsks
}

func (n *peerNetwork) documentsHeldBy(address string, word yacymodel.Hash) []yacymodel.URLHash {
	for spelledWord, addresses := range n.documentsPerWordPerPeer[address] {
		if yacymodel.WordHash(spelledWord) != word {
			continue
		}

		return documentHashesOf(addresses)
	}

	return nil
}

func (n *peerNetwork) AskForURLMetadata(
	_ context.Context,
	asks []peerasks.URLMetadataAsk,
) []peerasks.AnsweredURLMetadataAsk {
	n.urlMetadataAsks = append(n.urlMetadataAsks, asks...)

	answeredAsks := make([]peerasks.AnsweredURLMetadataAsk, 0, len(asks))
	for _, ask := range asks {
		answeredAsks = append(answeredAsks, peerasks.AnsweredURLMetadataAsk{
			Ask:   ask,
			Items: itemsOf(ask.Documents),
		})
	}

	return answeredAsks
}

func documentHashesOf(addresses []string) []yacymodel.URLHash {
	hashes := make([]yacymodel.URLHash, 0, len(addresses))
	for _, address := range addresses {
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			continue
		}
		hashes = append(hashes, hash)
	}

	return hashes
}

func itemsOf(documents []yacymodel.URLHash) []searchresult.Item {
	items := make([]searchresult.Item, 0, len(documents))
	for _, document := range documents {
		items = append(items, searchresult.Item{Hash: document})
	}

	return items
}

type responsiblePeers struct {
	peersPerWord map[string][]string
}

func (r responsiblePeers) ChoosePeersPerQueryWord(
	_ context.Context,
	queryWords []yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
	_ int,
) [][]peerdirectory.AskablePeer {
	peersPerQueryWord := make([][]peerdirectory.AskablePeer, 0, len(queryWords))
	for _, queryWord := range queryWords {
		peersPerQueryWord = append(peersPerQueryWord, r.peersForWord(queryWord, askablePeers))
	}

	return peersPerQueryWord
}

func (r responsiblePeers) peersForWord(
	word yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	if r.peersPerWord == nil {
		return askablePeers
	}
	for spelledWord, addresses := range r.peersPerWord {
		if yacymodel.WordHash(spelledWord) != word {
			continue
		}

		return peersAt(addresses)
	}

	return nil
}

func peersAt(addresses []string) []peerdirectory.AskablePeer {
	peers := make([]peerdirectory.AskablePeer, 0, len(addresses))
	for _, address := range addresses {
		peers = append(peers, peerAt(address))
	}

	return peers
}

type recordedSearches struct {
	performed []wordjoined.PerformedWordJoinedSearch
}

func (r *recordedSearches) WordJoinedSearchPerformed(
	_ context.Context,
	search wordjoined.PerformedWordJoinedSearch,
) {
	r.performed = append(r.performed, search)
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash(address), Address: address}
}

func distinctDocumentsAskedMetadataFor(asks []peerasks.URLMetadataAsk) []yacymodel.URLHash {
	documentsToAskMetadataFor := map[yacymodel.URLHash]struct{}{}
	for _, ask := range asks {
		for _, document := range ask.Documents {
			documentsToAskMetadataFor[document] = struct{}{}
		}
	}

	documents := make([]yacymodel.URLHash, 0, len(documentsToAskMetadataFor))
	for document := range documentsToAskMetadataFor {
		documents = append(documents, document)
	}
	slices.SortFunc(documents, func(first, second yacymodel.URLHash) int {
		return strings.Compare(first.String(), second.String())
	})

	return documents
}

func searchOf(
	network *peerNetwork,
	observer wordjoined.WordJoinedSearchObserver,
) [][]searchresult.Item {
	return searchChoosing(network, responsiblePeers{}, observer)
}

func searchChoosing(
	network *peerNetwork,
	choice responsiblePeers,
	observer wordjoined.WordJoinedSearchObserver,
) [][]searchresult.Item {
	return searchAskingMetadataForUpTo(network, choice, documentsToAskMetadataForCeiling, observer)
}

func searchAskingMetadataForUpTo(
	network *peerNetwork,
	choice responsiblePeers,
	documentsToAskMetadataForCeiling int,
	observer wordjoined.WordJoinedSearchObserver,
) [][]searchresult.Item {
	return searchOverPeers(
		wordjoined.New(
			network,
			choice,
			documentsToAskMetadataForCeiling,
			peerCallsCeiling,
			observer,
		),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	)
}

func searchOverPeers(
	spread wordjoined.Spread,
	askablePeers []peerdirectory.AskablePeer,
) [][]searchresult.Item {
	return spread.SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom(firstWord+" "+secondWord),
		askablePeers,
	)
}

func TestOnlyDocumentsThatEveryQueryWordCameBackForAreAskedAbout(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first": {
			firstWord: {"https://shared.example/", "https://only-first.example/"},
		},
		"second": {
			secondWord: {"https://shared.example/", "https://only-second.example/"},
		},
	})

	searchOf(network, &recordedSearches{})

	wanted := documentHashesOf([]string{"https://shared.example/"})
	if got := distinctDocumentsAskedMetadataFor(
		network.urlMetadataAsks,
	); !slices.Equal(
		got,
		wanted,
	) {
		t.Fatalf("asked about %v, want only the document both words came back for", got)
	}
}

func TestEachPeerIsAskedOnlyAboutTheDocumentsItHolds(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first": {
			firstWord:  {"https://shared.example/"},
			secondWord: {"https://shared.example/"},
		},
		"second": {
			firstWord:  {"https://other.example/"},
			secondWord: {"https://other.example/"},
		},
	})

	searchOf(network, &recordedSearches{})

	for _, ask := range network.urlMetadataAsks {
		held := network.documentsPerWordPerPeer[ask.Peer.Address][firstWord]
		if !slices.Equal(ask.Documents, documentHashesOf(held)) {
			t.Fatalf(
				"peer %q was asked about %v, want only %v",
				ask.Peer.Address,
				ask.Documents,
				held,
			)
		}
	}
}

func TestAPeerHoldingOnlyOneQueryWordIsStillAskedAboutAJoinedDocument(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://shared.example/"}},
		"second": {secondWord: {"https://shared.example/"}},
	})

	searchOf(network, &recordedSearches{})

	if len(network.urlMetadataAsks) != 2 {
		t.Fatalf(
			"%d peers were asked about the joined document, want both holders",
			len(network.urlMetadataAsks),
		)
	}
	for _, ask := range network.urlMetadataAsks {
		if len(ask.Documents) != 1 {
			t.Fatalf(
				"peer %q was asked about %v, want the one joined document",
				ask.Peer.Address,
				ask.Documents,
			)
		}
	}
}

func TestNoPeerIsAskedAboutADocumentWhenNoDocumentIsHeldForEveryWord(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://only-first.example/"}},
		"second": {secondWord: {"https://only-second.example/"}},
	})

	items := searchOf(network, &recordedSearches{})

	if len(network.urlMetadataAsks) != 0 || len(items) != 0 {
		t.Fatalf(
			"asked %d peers and answered %d peers of items, want none of either",
			len(network.urlMetadataAsks),
			len(items),
		)
	}
}

func TestOnlyThePeersResponsibleForAWordAreAskedWhatTheyHoldForIt(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://shared.example/"}},
		"second": {secondWord: {"https://shared.example/"}},
	})
	observer := &recordedSearches{}

	searchChoosing(network, responsiblePeers{peersPerWord: map[string][]string{
		firstWord:  {"first"},
		secondWord: {"second"},
	}}, observer)

	for _, ask := range network.heldDocumentsAsks {
		if ask.Peer.Address == "first" && ask.Word != yacymodel.WordHash(firstWord) {
			t.Fatalf("peer %q was asked what it holds for a word it is not responsible for",
				ask.Peer.Address)
		}
		if ask.Peer.Address == "second" && ask.Word != yacymodel.WordHash(secondWord) {
			t.Fatalf("peer %q was asked what it holds for a word it is not responsible for",
				ask.Peer.Address)
		}
	}
	if len(network.heldDocumentsAsks) != 2 {
		t.Fatalf(
			"%d held documents asks were put, want one for each responsible peer",
			len(network.heldDocumentsAsks),
		)
	}
}

func TestTheSearchReportsWhatEveryQueryWordWasHeldFor(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first": {
			firstWord: {"https://shared.example/", "https://only-first.example/"},
		},
		"second": {
			secondWord: {"https://shared.example/"},
		},
	})
	network.silentPeers["never"] = struct{}{}
	observer := &recordedSearches{}

	searchOf(network, observer)

	if len(observer.performed) != 1 {
		t.Fatalf("the observer saw %d searches, want one", len(observer.performed))
	}
	performed := observer.performed[0]
	if performed.AmountOfQueryWords != 2 || performed.AmountOfPeersAskedForHeldDocuments != 2 ||
		performed.AmountOfJoinedDocuments != 1 {
		t.Fatalf("the search reported %+v, want two words, two peers and one joined document",
			performed)
	}
	if performed.AmountOfQueryWordsNoPeerHeld != 0 ||
		performed.AmountOfDocumentsThatCameBack != 1 {
		t.Fatalf(
			"the search reported %+v, want every word held and the joined document back",
			performed,
		)
	}
}

func TestAQueryWordNoPeerHeldIsReported(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://only-first.example/"}},
		"second": {firstWord: {"https://only-first.example/"}},
	})
	observer := &recordedSearches{}

	searchOf(network, observer)

	if observer.performed[0].AmountOfQueryWordsNoPeerHeld != 1 {
		t.Fatalf(
			"the search reported %d query words no peer held, want the one word nobody held",
			observer.performed[0].AmountOfQueryWordsNoPeerHeld,
		)
	}
}

func TestAPeerThatDoesNotAnswerHoldsNothingForTheJoin(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://shared.example/"}},
		"second": {secondWord: {"https://shared.example/"}},
	})
	network.silentPeers["second"] = struct{}{}
	observer := &recordedSearches{}

	searchOf(network, observer)

	if len(network.urlMetadataAsks) != 0 {
		t.Fatalf(
			"asked %d peers about documents, want none once a word went unanswered",
			len(network.urlMetadataAsks),
		)
	}
	if observer.performed[0].AmountOfPeersThatNamedHeldDocuments != 1 {
		t.Fatalf(
			"the search reported %d answering peers, want one",
			observer.performed[0].AmountOfPeersThatNamedHeldDocuments,
		)
	}
}

func TestNoMoreDocumentsThanTheCeilingAreAskedMetadataFor(t *testing.T) {
	t.Parallel()

	joined := []string{
		"https://first.example/",
		"https://second.example/",
		"https://third.example/",
	}
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: joined, secondWord: joined},
	})

	searchAskingMetadataForUpTo(network, responsiblePeers{}, 1, &recordedSearches{})

	if got := distinctDocumentsAskedMetadataFor(network.urlMetadataAsks); len(got) != 1 {
		t.Fatalf(
			"the search asked metadata for %d documents, want the one the ceiling allows",
			len(got),
		)
	}
}

func TestTheDocumentsTheMostPeersHoldAreTheOnesAskedMetadataFor(t *testing.T) {
	t.Parallel()

	first := "https://first.example/"
	second := "https://second.example/"

	if got := theOneDocumentAskedMetadataFor(t, first, second); got != documentHashOf(t, first) {
		t.Fatalf("%v went out to be matched, want the document both peers hold", got)
	}
	if got := theOneDocumentAskedMetadataFor(t, second, first); got != documentHashOf(t, second) {
		t.Fatalf("%v went out to be matched, want the document both peers hold", got)
	}
}

func theOneDocumentAskedMetadataFor(
	t *testing.T,
	heldByBothPeers string,
	heldByOnePeer string,
) yacymodel.URLHash {
	t.Helper()

	joined := []string{heldByBothPeers, heldByOnePeer}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: joined, secondWord: joined},
		"second": {firstWord: {heldByBothPeers}, secondWord: {heldByBothPeers}},
	})

	searchAskingMetadataForUpTo(network, responsiblePeers{}, 1, &recordedSearches{})

	documents := distinctDocumentsAskedMetadataFor(network.urlMetadataAsks)
	if len(documents) != 1 {
		t.Fatalf("%d documents went out to be matched, want the one the ceiling allows",
			len(documents))
	}

	return documents[0]
}

func documentHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hashes := documentHashesOf([]string{address})
	if len(hashes) != 1 {
		t.Fatalf("%q has no document hash", address)
	}

	return hashes[0]
}

func TestTheSearchReportsTheWholeJoinBesideTheDocumentsItAskedMetadataFor(t *testing.T) {
	t.Parallel()

	joined := []string{"https://first.example/", "https://second.example/"}
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: joined, secondWord: joined},
	})
	observer := &recordedSearches{}

	searchAskingMetadataForUpTo(network, responsiblePeers{}, 1, observer)

	performed := observer.performed[0]
	if performed.AmountOfJoinedDocuments != 2 || performed.AmountOfDocumentsToAskMetadataFor != 1 {
		t.Fatalf(
			"the search reported %+v, want two joined documents and one asked metadata for",
			performed,
		)
	}
	if performed.AmountOfDocumentsThatCameBack != 1 {
		t.Fatalf(
			"the search reported %d documents back, want the one it asked metadata for",
			performed.AmountOfDocumentsThatCameBack,
		)
	}
}

func TestNoMorePeersAreAskedForMetadataThanTheCallsOneQueryMayPut(t *testing.T) {
	t.Parallel()

	const peerCalls = 2

	bothDocuments := []string{"https://first.example/", "https://second.example/"}
	firstDocument, secondDocument := bothDocuments[:1], bothDocuments[1:]
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: firstDocument, secondWord: firstDocument},
		"second": {firstWord: secondDocument, secondWord: secondDocument},
		"third":  {firstWord: firstDocument, secondWord: firstDocument},
		"fourth": {firstWord: secondDocument, secondWord: secondDocument},
	})

	searchOverPeers(
		wordjoined.New(
			network,
			responsiblePeers{},
			documentsToAskMetadataForCeiling,
			peerCalls,
			&recordedSearches{},
		),
		peersAt([]string{"first", "second", "third", "fourth"}),
	)

	if len(network.urlMetadataAsks) != peerCalls {
		t.Fatalf(
			"%d peers were asked for metadata, want %d",
			len(network.urlMetadataAsks),
			peerCalls,
		)
	}
	wanted := documentHashesOf(bothDocuments)
	got := distinctDocumentsAskedMetadataFor(network.urlMetadataAsks)
	slices.SortFunc(wanted, func(first, second yacymodel.URLHash) int {
		return strings.Compare(first.String(), second.String())
	})
	if !slices.Equal(got, wanted) {
		t.Fatalf("asked about %v, want every joined document %v", got, wanted)
	}
}
