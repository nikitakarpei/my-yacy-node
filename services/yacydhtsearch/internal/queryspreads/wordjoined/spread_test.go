package wordjoined_test

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord                = "berlin"
	secondWord               = "weather"
	metadataDocumentsCeiling = 10
	peerItemsCeiling         = 10
	peersHoldingOneWord      = 24
)

type peerNetwork struct {
	documentsPerWordPerPeer     map[string]map[string][]string
	answeredItemsPerWordPerPeer map[string]map[string][]string
	countsAWordWithEachItem     bool
	documentsHeldForEveryWord   int
	peersCountingNoDocument     map[string]struct{}
	heldDocumentsAsks           []peerasks.HeldDocumentsAsk
	urlMetadataAsks             []peerasks.URLMetadataAsk
	silentPeers                 map[string]struct{}
}

func networkOf(documentsPerWordPerPeer map[string]map[string][]string) *peerNetwork {
	return &peerNetwork{
		documentsPerWordPerPeer: documentsPerWordPerPeer,
		peersCountingNoDocument: map[string]struct{}{},
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
			Ask:                     ask,
			DocumentsHeldForTheWord: n.documentsHeldBy(ask.Peer.Address, ask.Word),
			MatchedDocuments: n.matchedDocumentsOf(
				documentsPerWordOf(n.answeredItemsPerWordPerPeer, ask.Peer.Address, ask.Word),
			),
			AmountOfDocumentsHeldForTheWord: n.documentsCountedBy(ask.Peer.Address),
		})
	}

	return answeredAsks
}

func (n *peerNetwork) matchedDocumentsOf(
	documents []yacymodel.URLHash,
) []peerasks.MatchedDocument {
	matchedDocuments := make([]peerasks.MatchedDocument, 0, len(documents))
	for _, document := range documents {
		matchedDocument := peerasks.MatchedDocument{
			Metadata: yacymodel.URLMetadata{Hash: document},
		}
		if n.countsAWordWithEachItem {
			matchedDocument.CountOfAWordTheAskNamed = peeranswers.WordCount{Hits: 3}
		}
		matchedDocuments = append(matchedDocuments, matchedDocument)
	}

	return matchedDocuments
}

func (n *peerNetwork) documentsCountedBy(address string) yacymodel.Optional[int] {
	if _, countsNoDocument := n.peersCountingNoDocument[address]; countsNoDocument {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(n.documentsHeldForEveryWord)
}

func (n *peerNetwork) documentsHeldBy(address string, word yacymodel.Hash) []yacymodel.URLHash {
	return documentsPerWordOf(n.documentsPerWordPerPeer, address, word)
}

func documentsPerWordOf(
	documentsPerWordPerPeer map[string]map[string][]string,
	address string,
	word yacymodel.Hash,
) []yacymodel.URLHash {
	for spelledWord, addresses := range documentsPerWordPerPeer[address] {
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
			Ask:                    ask,
			MetadataOfEachDocument: metadataOfEachDocument(ask.Documents),
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

func metadataOfEachDocument(documents []yacymodel.URLHash) []yacymodel.URLMetadata {
	metadataOfEachDocument := make([]yacymodel.URLMetadata, 0, len(documents))
	for _, document := range documents {
		metadataOfEachDocument = append(metadataOfEachDocument, yacymodel.URLMetadata{
			Hash: document,
		})
	}

	return metadataOfEachDocument
}

type responsiblePeers struct {
	peerAddressesPerWord map[string][]string
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
	if r.peerAddressesPerWord == nil {
		return askablePeers
	}
	for spelledWord, addresses := range r.peerAddressesPerWord {
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

type recordedSpreads struct {
	performed []wordjoined.PerformedWordJoinedSpread
}

func (r *recordedSpreads) WordJoinedSpreadPerformed(
	_ context.Context,
	spread wordjoined.PerformedWordJoinedSpread,
) {
	r.performed = append(r.performed, spread)
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{
		Hash:    yacymodel.WordHash(address),
		Address: address,
	}
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

func spreadOf(
	network *peerNetwork,
	observer wordjoined.WordJoinedSpreadObserver,
) peeranswers.AnsweredQuery {
	return spreadChoosing(network, responsiblePeers{}, observer)
}

func spreadChoosing(
	network *peerNetwork,
	choice responsiblePeers,
	observer wordjoined.WordJoinedSpreadObserver,
) peeranswers.AnsweredQuery {
	return spreadAskingMetadataForUpTo(network, choice, metadataDocumentsCeiling, observer)
}

func spreadAskingMetadataForUpTo(
	network *peerNetwork,
	choice responsiblePeers,
	metadataDocumentsCeiling int,
	observer wordjoined.WordJoinedSpreadObserver,
) peeranswers.AnsweredQuery {
	return spreadOverPeers(
		wordjoined.New(
			network,
			choice,
			metadataDocumentsCeiling,
			peerItemsCeiling,
			peersHoldingOneWord,
			observer,
		),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	)
}

func spreadOverPeers(
	spread wordjoined.Spread,
	askablePeers []peerdirectory.AskablePeer,
) peeranswers.AnsweredQuery {
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

	spreadOf(network, &recordedSpreads{})

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

	spreadOf(network, &recordedSpreads{})

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

	spreadOf(network, &recordedSpreads{})

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

	items := spreadOf(network, &recordedSpreads{}).ItemsInTheOrderOfEachAnswer

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
	observer := &recordedSpreads{}

	spreadChoosing(network, responsiblePeers{peerAddressesPerWord: map[string][]string{
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

func TestTheSpreadReportsWhatEveryQueryWordWasHeldFor(t *testing.T) {
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
	observer := &recordedSpreads{}

	spreadOf(network, observer)

	if len(observer.performed) != 1 {
		t.Fatalf("the observer saw %d spreads, want one", len(observer.performed))
	}
	performed := observer.performed[0]
	if performed.AmountOfQueryWords != 2 || performed.AmountOfPeersAsked != 2 ||
		performed.AmountOfJoinedDocuments != 1 {
		t.Fatalf("the spread reported %+v, want two words, two peers and one joined document",
			performed)
	}
	if performed.AmountOfQueryWordsNoPeerHeld != 0 ||
		performed.AmountOfAskedDocumentsWithMetadata != 1 {
		t.Fatalf(
			"the spread reported %+v, want every word held and the joined document back",
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
	observer := &recordedSpreads{}

	spreadOf(network, observer)

	if observer.performed[0].AmountOfQueryWordsNoPeerHeld != 1 {
		t.Fatalf(
			"the spread reported %d query words no peer held, want the one word nobody held",
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
	observer := &recordedSpreads{}

	spreadOf(network, observer)

	if len(network.urlMetadataAsks) != 0 {
		t.Fatalf(
			"asked %d peers about documents, want none once a word went unanswered",
			len(network.urlMetadataAsks),
		)
	}
	if observer.performed[0].AmountOfPeersThatAnswered != 1 {
		t.Fatalf(
			"the spread reported %d answering peers, want one",
			observer.performed[0].AmountOfPeersThatAnswered,
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

	spreadAskingMetadataForUpTo(network, responsiblePeers{}, 1, &recordedSpreads{})

	if got := distinctDocumentsAskedMetadataFor(network.urlMetadataAsks); len(got) != 1 {
		t.Fatalf(
			"the spread asked metadata for %d documents, want the one the ceiling allows",
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

	spreadAskingMetadataForUpTo(network, responsiblePeers{}, 1, &recordedSpreads{})

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

func TestTheSpreadReportsTheWholeJoinBesideTheDocumentsItAskedMetadataFor(t *testing.T) {
	t.Parallel()

	joined := []string{"https://first.example/", "https://second.example/"}
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: joined, secondWord: joined},
	})
	observer := &recordedSpreads{}

	spreadAskingMetadataForUpTo(network, responsiblePeers{}, 1, observer)

	performed := observer.performed[0]
	if performed.AmountOfJoinedDocuments != 2 || performed.AmountOfDocumentsAskedMetadataFor != 1 {
		t.Fatalf(
			"the spread reported %+v, want two joined documents and one asked metadata for",
			performed,
		)
	}
	if performed.AmountOfAskedDocumentsWithMetadata != 1 {
		t.Fatalf(
			"the spread reported %d documents back, want the one it asked metadata for",
			performed.AmountOfAskedDocumentsWithMetadata,
		)
	}
}

func TestNoMorePeersAreAskedForMetadataThanHoldOneWord(t *testing.T) {
	t.Parallel()

	const peersOfOneWord = 2

	bothDocuments := []string{"https://first.example/", "https://second.example/"}
	firstDocument, secondDocument := bothDocuments[:1], bothDocuments[1:]
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: firstDocument, secondWord: firstDocument},
		"second": {firstWord: secondDocument, secondWord: secondDocument},
		"third":  {firstWord: firstDocument, secondWord: firstDocument},
		"fourth": {firstWord: secondDocument, secondWord: secondDocument},
	})

	spreadOverPeers(
		wordjoined.New(
			network,
			responsiblePeers{},
			metadataDocumentsCeiling,
			peerItemsCeiling,
			peersOfOneWord,
			&recordedSpreads{},
		),
		peersAt([]string{"first", "second", "third", "fourth"}),
	)

	if len(network.urlMetadataAsks) != peersOfOneWord {
		t.Fatalf(
			"%d peers were asked for metadata, want %d",
			len(network.urlMetadataAsks),
			peersOfOneWord,
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

func TestAJoinedDocumentAPeerAlreadyAnsweredIsNotAskedMetadataFor(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	unanswered := "https://unanswered.example/"
	joined := []string{answered, unanswered}
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: joined, secondWord: joined},
	})
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"first": {firstWord: {answered}},
	}

	spreadOf(network, &recordedSpreads{})

	wanted := documentHashesOf([]string{unanswered})
	if got := distinctDocumentsAskedMetadataFor(network.urlMetadataAsks); !slices.Equal(
		got, wanted,
	) {
		t.Fatalf("asked about %v, want only the joined document no peer answered", got)
	}
}

func TestTheItemsAPeerAnsweredForJoinedDocumentsComeBack(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	})
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"first": {firstWord: {answered, "https://unjoined.example/"}},
	}

	itemsInTheOrderOfEachAnswer := spreadOf(network, &recordedSpreads{}).ItemsInTheOrderOfEachAnswer

	wanted := documentHashOf(t, answered)
	documents := map[yacymodel.URLHash]struct{}{}
	for _, items := range itemsInTheOrderOfEachAnswer {
		for _, item := range items {
			documents[item.Metadata.Hash] = struct{}{}
		}
	}
	if _, cameBack := documents[wanted]; !cameBack || len(documents) != 1 {
		t.Fatalf("the spread answered %v, want only the joined document the peer answered",
			documents)
	}
}

func TestEveryAnsweredItemMatchedEveryQueryWord(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	fetched := "https://fetched.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered, fetched}, secondWord: {answered, fetched}},
	})
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"first": {firstWord: {answered}},
	}

	answers := spreadOf(network, &recordedSpreads{})

	wantedWords := []yacymodel.Hash{
		yacymodel.WordHash(firstWord), yacymodel.WordHash(secondWord),
	}
	itemsOfEveryAnswer := slices.Concat(
		answers.ItemsInTheOrderOfEachAnswer, [][]peeranswers.AnsweredItem{answers.ItemsInNoOrder},
	)
	for _, items := range itemsOfEveryAnswer {
		for _, item := range items {
			for _, word := range wantedWords {
				if _, matched := item.MatchedWords[word]; !matched {
					t.Fatalf("the item of %v matched %v, want every query word",
						item.Metadata.Hash, item.MatchedWords)
				}
			}
		}
	}
}

func TestAnAnsweredItemIsCountedForTheWordThePeerWasAskedAbout(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	})
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"first": {firstWord: {answered}},
	}
	network.countsAWordWithEachItem = true

	itemsInTheOrderOfEachAnswer := spreadOf(network, &recordedSpreads{}).ItemsInTheOrderOfEachAnswer

	if len(itemsInTheOrderOfEachAnswer) != 1 || len(itemsInTheOrderOfEachAnswer[0]) != 1 {
		t.Fatalf(
			"the spread answered %v, want the one item the peer answered",
			itemsInTheOrderOfEachAnswer,
		)
	}
	matchedWords := itemsInTheOrderOfEachAnswer[0][0].MatchedWords
	if matchedWords[yacymodel.WordHash(firstWord)].Hits != 3 ||
		matchedWords[yacymodel.WordHash(secondWord)].CountedByAPeer() {
		t.Fatalf("the item matched %v, want the count under the word the ask named", matchedWords)
	}
}

func TestTheAnswersCarryTheDocumentsTheNetworkHoldsForEachQueryWord(t *testing.T) {
	t.Parallel()

	joined := []string{"https://joined.example/"}
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: joined, secondWord: joined},
	})
	network.documentsHeldForEveryWord = 512

	answers := spreadOf(network, &recordedSpreads{})

	want := map[yacymodel.Hash]int{
		yacymodel.WordHash(firstWord):  2 * 512,
		yacymodel.WordHash(secondWord): 2 * 512,
	}
	if got := answers.DocumentsHeldPerQueryWord; !maps.Equal(got, want) {
		t.Fatalf("the answers carry %v documents per query word, want %v", got, want)
	}
}

func TestAPeerThatCountsNoDocumentForAWordSaysNothingOfWhatTheNetworkHolds(t *testing.T) {
	t.Parallel()

	joined := []string{"https://joined.example/"}
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: joined, secondWord: joined},
	})
	network.documentsHeldForEveryWord = 512
	network.peersCountingNoDocument = map[string]struct{}{"second": {}}

	answers := spreadOf(network, &recordedSpreads{})

	want := map[yacymodel.Hash]int{
		yacymodel.WordHash(firstWord):  512,
		yacymodel.WordHash(secondWord): 512,
	}
	if got := answers.DocumentsHeldPerQueryWord; !maps.Equal(got, want) {
		t.Fatalf("the answers carry %v documents per query word, want %v", got, want)
	}
}

func TestTheSpreadReportsWhatThePeersAnsweredBesideTheDocumentsTheyHold(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	})
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	}
	network.countsAWordWithEachItem = true
	network.documentsHeldForEveryWord = 512
	observer := &recordedSpreads{}

	spreadChoosing(network, responsiblePeers{peerAddressesPerWord: map[string][]string{
		firstWord:  {"first"},
		secondWord: {"first"},
	}}, observer)

	performed := observer.performed[0]
	if performed.AmountOfJoinedDocumentsWithMetadata != 1 ||
		performed.AmountOfMatchedDocumentsAcrossAnswers != 2 ||
		performed.AmountOfMatchedDocumentsCountedByAPeer != 2 {
		t.Fatalf(
			"the spread reported %+v, want the joined document answered once and two counts",
			performed,
		)
	}
	if !slices.Equal(performed.AmountOfDocumentsHeldInEachAnswer, []int{512, 512}) {
		t.Fatalf(
			"the spread reported %v documents held per query word, want 512 for each answer",
			performed.AmountOfDocumentsHeldInEachAnswer,
		)
	}
}

func TestTheNearestPeerOfEveryWordIsAskedBeforeTheNextPeerOfAnyWord(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{})

	spreadChoosing(network, responsiblePeers{peerAddressesPerWord: map[string][]string{
		firstWord:  {"nearest-of-first", "next-of-first"},
		secondWord: {"nearest-of-second", "next-of-second"},
	}}, &recordedSpreads{})

	wanted := []string{
		firstWord + " nearest-of-first",
		secondWord + " nearest-of-second",
		firstWord + " next-of-first",
		secondWord + " next-of-second",
	}
	if got := wordsAndPeersAskedInOrder(network.heldDocumentsAsks); !slices.Equal(got, wanted) {
		t.Fatalf("the spread asked %v, want a turn of every word before the next peer", got)
	}
}

func wordsAndPeersAskedInOrder(asks []peerasks.HeldDocumentsAsk) []string {
	wordsAndPeers := make([]string, 0, len(asks))
	for _, ask := range asks {
		wordsAndPeers = append(wordsAndPeers, spelledWordOf(ask.Word)+" "+ask.Peer.Address)
	}

	return wordsAndPeers
}

func spelledWordOf(word yacymodel.Hash) string {
	for _, spelledWord := range []string{firstWord, secondWord} {
		if yacymodel.WordHash(spelledWord) == word {
			return spelledWord
		}
	}

	return word.String()
}
