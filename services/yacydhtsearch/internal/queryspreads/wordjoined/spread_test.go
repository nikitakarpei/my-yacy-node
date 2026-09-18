package wordjoined_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord                      = "berlin"
	secondWord                     = "weather"
	thirdWord                      = "rain"
	metadataDocumentsCeiling       = 10
	crossCheckedDocumentsCeiling   = 10
	asksForCrossCheckedDocuments   = true
	asksForNoCrossCheckedDocuments = false
	peerItemsCeiling               = 10
	peersHoldingOneWord            = 24
	onePartitionOfTheRing          = 1
	twoPartitionsOfTheRing         = 2
)

type peerNetwork struct {
	documentsPerWordPerPeer               map[string]map[string][]string
	answeredItemsPerWordPerPeer           map[string]map[string][]string
	countsAWordWithEachItem               bool
	documentsHeldForEveryWord             int
	documentsHeldByEachPeer               map[string]int
	documentsPerAnswerOfEachPeer          map[string]int
	peersCountingNoDocument               map[string]struct{}
	matchedAndHeldDocumentsAsks           []peerasks.MatchedAndHeldDocumentsAsk
	crossCheckedDocumentsAsks             []peerasks.CrossCheckedDocumentsAsk
	urlMetadataAsks                       []peerasks.URLMetadataAsk
	silentPeers                           map[string]struct{}
	peersSilentInTheCrossCheckedDocuments map[string]struct{}
	peersHoldingNoNamedDocument           map[string]struct{}
	timeLeftInEachRoundInTheirOrder       []time.Duration
}

func networkOf(documentsPerWordPerPeer map[string]map[string][]string) *peerNetwork {
	return &peerNetwork{
		documentsPerWordPerPeer:               documentsPerWordPerPeer,
		peersCountingNoDocument:               map[string]struct{}{},
		silentPeers:                           map[string]struct{}{},
		peersSilentInTheCrossCheckedDocuments: map[string]struct{}{},
		peersHoldingNoNamedDocument:           map[string]struct{}{},
	}
}

func (n *peerNetwork) AskForMatchedAndHeldDocuments(
	ctx context.Context,
	asks []peerasks.MatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	n.matchedAndHeldDocumentsAsks = append(n.matchedAndHeldDocumentsAsks, asks...)
	n.recordTimeLeftIn(ctx)

	answeredAsks := make([]peerasks.AnsweredMatchedAndHeldDocumentsAsk, 0, len(asks))
	for _, ask := range asks {
		if _, silent := n.silentPeers[ask.Peer.Address]; silent {
			continue
		}
		answeredAsks = append(answeredAsks, peerasks.AnsweredMatchedAndHeldDocumentsAsk{
			Ask:                       ask,
			DocumentsListedForTheWord: n.documentsListedBy(ask.Peer.Address, ask.Word),
			MatchedDocuments: n.matchedDocumentsOf(
				documentsPerWordOf(n.answeredItemsPerWordPerPeer, ask.Peer.Address, ask.Word),
			),
			AmountOfDocumentsHeldForTheWord: n.documentsCountedBy(ask.Peer.Address, ask.Word),
		})
	}

	return answeredAsks
}

func (n *peerNetwork) AskForCrossCheckedDocuments(
	ctx context.Context,
	asks []peerasks.CrossCheckedDocumentsAsk,
) []peerasks.AnsweredCrossCheckedDocumentsAsk {
	n.crossCheckedDocumentsAsks = append(n.crossCheckedDocumentsAsks, asks...)
	n.recordTimeLeftIn(ctx)

	answeredAsks := make([]peerasks.AnsweredCrossCheckedDocumentsAsk, 0, len(asks))
	for _, ask := range asks {
		if _, silent := n.peersSilentInTheCrossCheckedDocuments[ask.Peer.Address]; silent {
			continue
		}
		answeredAsks = append(answeredAsks, peerasks.AnsweredCrossCheckedDocumentsAsk{
			Ask:                     ask,
			DocumentsHeldForTheWord: n.namedDocumentsHeldBy(ask),
		})
	}

	return answeredAsks
}

func (n *peerNetwork) namedDocumentsHeldBy(
	ask peerasks.CrossCheckedDocumentsAsk,
) []yacymodel.URLHash {
	if _, holdsNothing := n.peersHoldingNoNamedDocument[ask.Peer.Address]; holdsNothing {
		return nil
	}

	heldDocuments := documentsPerWordOf(n.documentsPerWordPerPeer, ask.Peer.Address, ask.Word)
	namedDocumentsHeld := make([]yacymodel.URLHash, 0, len(ask.Documents))
	for _, document := range ask.Documents {
		if !slices.Contains(heldDocuments, document) {
			continue
		}
		namedDocumentsHeld = append(namedDocumentsHeld, document)
	}

	return namedDocumentsHeld
}

func (n *peerNetwork) recordTimeLeftIn(ctx context.Context) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		n.timeLeftInEachRoundInTheirOrder = append(n.timeLeftInEachRoundInTheirOrder, 0)

		return
	}
	n.timeLeftInEachRoundInTheirOrder = append(
		n.timeLeftInEachRoundInTheirOrder, time.Until(deadline),
	)
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
			matchedDocument.Posting = yacymodel.Some(yacymodel.RWIPosting{Hits: 3})
		}
		matchedDocuments = append(matchedDocuments, matchedDocument)
	}

	return matchedDocuments
}

func (n *peerNetwork) documentsCountedBy(
	address string,
	word yacymodel.Hash,
) yacymodel.Optional[int] {
	if _, countsNoDocument := n.peersCountingNoDocument[address]; countsNoDocument {
		return yacymodel.None[int]()
	}
	if _, answersInPart := n.documentsPerAnswerOfEachPeer[address]; answersInPart {
		return yacymodel.Some(
			len(documentsPerWordOf(n.documentsPerWordPerPeer, address, word)),
		)
	}
	if documentsHeld, countsItsOwn := n.documentsHeldByEachPeer[address]; countsItsOwn {
		return yacymodel.Some(documentsHeld)
	}

	return yacymodel.Some(n.documentsHeldForEveryWord)
}

func (n *peerNetwork) documentsListedBy(address string, word yacymodel.Hash) []yacymodel.URLHash {
	documents := documentsPerWordOf(n.documentsPerWordPerPeer, address, word)
	documentsPerAnswer, answersInPart := n.documentsPerAnswerOfEachPeer[address]
	if !answersInPart || len(documents) <= documentsPerAnswer {
		return documents
	}

	return documents[:documentsPerAnswer]
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
	ctx context.Context,
	asks []peerasks.URLMetadataAsk,
) []peerasks.AnsweredURLMetadataAsk {
	n.urlMetadataAsks = append(n.urlMetadataAsks, asks...)
	n.recordTimeLeftIn(ctx)

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
	partitionOfEachPeer  map[string]uint
}

func (r responsiblePeers) ChosenPeersPerQueryWordFor(
	_ context.Context,
	queryWords []yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
) peerchoice.ChosenPeersPerQueryWord {
	peersPerQueryWord := make(peerchoice.ChosenPeersPerQueryWord, 0, len(queryWords))
	for _, queryWord := range queryWords {
		peersPerQueryWord = append(peersPerQueryWord, peerchoice.ChosenPeersOfQueryWord{
			QueryWord:   queryWord,
			ChosenPeers: r.chosenPeersOf(r.peersForWord(queryWord, askablePeers)),
		})
	}

	return peersPerQueryWord
}

func (r responsiblePeers) chosenPeersOf(
	askablePeers []peerdirectory.AskablePeer,
) []peerchoice.ChosenPeer {
	chosenPeers := make([]peerchoice.ChosenPeer, 0, len(askablePeers))
	for _, peer := range askablePeers {
		chosenPeers = append(chosenPeers, peerchoice.ChosenPeer{
			Peer:      peer,
			Partition: r.partitionOfEachPeer[peer.Address],
		})
	}

	return chosenPeers
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
) queryanswers.AnsweredQuery {
	return spreadChoosing(network, responsiblePeers{}, observer)
}

func spreadChoosing(
	network *peerNetwork,
	choice responsiblePeers,
	observer wordjoined.WordJoinedSpreadObserver,
) queryanswers.AnsweredQuery {
	return spreadAskingMetadataForUpTo(network, choice, metadataDocumentsCeiling, observer)
}

func spreadAskingMetadataForUpTo(
	network *peerNetwork,
	choice responsiblePeers,
	metadataDocumentsCeiling int,
	observer wordjoined.WordJoinedSpreadObserver,
) queryanswers.AnsweredQuery {
	return spreadOverPeers(
		spreadChoosingPeersBy(
			choice,
			wordjoined.New(
				network,
				network,
				metadataDocumentsCeiling,
				asksForCrossCheckedDocuments,
				crossCheckedDocumentsCeiling,
				peerItemsCeiling,
				onePartitionOfTheRing,
				peersHoldingOneWord,
				observer,
			),
		),
		[]peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
	)
}

func spreadOverPeers(
	spread spreadChoosingPeers,
	askablePeers []peerdirectory.AskablePeer,
) queryanswers.AnsweredQuery {
	return spread.SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom(firstWord+" "+secondWord, ""),
		askablePeers,
	)
}

func spreadNamingCrossCheckedDocumentsForUpTo(
	network *peerNetwork,
	choice responsiblePeers,
	crossCheckedDocumentsCeiling int,
	observer wordjoined.WordJoinedSpreadObserver,
) {
	spreadOverPeers(
		spreadChoosingPeersBy(
			choice,
			wordjoined.New(
				network,
				network,
				metadataDocumentsCeiling,
				asksForCrossCheckedDocuments,
				crossCheckedDocumentsCeiling,
				peerItemsCeiling,
				onePartitionOfTheRing,
				peersHoldingOneWord,
				observer,
			),
		),
		peersAt([]string{"first", "second"}),
	)
}

func spreadNotAskingForCrossCheckedDocuments(
	network *peerNetwork,
	choice responsiblePeers,
) {
	spreadOverPeers(
		spreadChoosingPeersBy(
			choice,
			wordjoined.New(
				network,
				network,
				metadataDocumentsCeiling,
				asksForNoCrossCheckedDocuments,
				crossCheckedDocumentsCeiling,
				peerItemsCeiling,
				onePartitionOfTheRing,
				peersHoldingOneWord,
				&recordedSpreads{},
			),
		),
		peersAt([]string{"first", "second"}),
	)
}

func spreadOfTheQuery(
	network *peerNetwork,
	choice responsiblePeers,
	query string,
	observer wordjoined.WordJoinedSpreadObserver,
) {
	spreadChoosingPeersBy(
		choice,
		wordjoined.New(
			network,
			network,
			metadataDocumentsCeiling,
			asksForCrossCheckedDocuments,
			crossCheckedDocumentsCeiling,
			peerItemsCeiling,
			onePartitionOfTheRing,
			peersHoldingOneWord,
			observer,
		),
	).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom(query, ""),
		peersAt([]string{"first", "second"}),
	)
}

func spreadWithin(network *peerNetwork, budget time.Duration) {
	ctx, endQuery := context.WithTimeout(context.Background(), budget)
	defer endQuery()

	spreadChoosingPeersBy(
		responsiblePeers{},
		wordjoined.New(
			network,
			network,
			metadataDocumentsCeiling,
			asksForCrossCheckedDocuments,
			crossCheckedDocumentsCeiling,
			peerItemsCeiling,
			onePartitionOfTheRing,
			peersHoldingOneWord,
			&recordedSpreads{},
		),
	).SpreadOverPeers(
		ctx,
		searchquery.QueryFrom(firstWord+" "+secondWord, ""),
		peersAt([]string{"first", "second"}),
	)
}

func peersOfEachQueryWord(
	peerAddressesPerWord map[string][]string,
) responsiblePeers {
	return responsiblePeers{peerAddressesPerWord: peerAddressesPerWord}
}

func documentsAskedToCrossCheck(asks []peerasks.CrossCheckedDocumentsAsk) []yacymodel.URLHash {
	documents := make([]yacymodel.URLHash, 0, len(asks))
	for _, ask := range asks {
		documents = append(documents, ask.Documents...)
	}

	return documents
}

func documentsInTheirHashOrder(documents []yacymodel.URLHash) []yacymodel.URLHash {
	documentsInOrder := slices.Clone(documents)
	slices.SortFunc(documentsInOrder, func(first, second yacymodel.URLHash) int {
		return strings.Compare(first.String(), second.String())
	})

	return documentsInOrder
}

func TestASpreadThatAsksForNoCrossCheckedDocumentsLeavesTheSecondRoundOut(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://anchored.example/", "https://answered.example/"}},
		"second": {secondWord: {"https://answered.example/", "https://anchored.example/"}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1}

	spreadNotAskingForCrossCheckedDocuments(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second"},
		}),
	)

	if len(network.crossCheckedDocumentsAsks) != 0 || len(network.urlMetadataAsks) == 0 {
		t.Fatalf(
			"the spread put %d cross-checked documents asks and %d metadata asks, want none and some",
			len(network.crossCheckedDocumentsAsks),
			len(network.urlMetadataAsks),
		)
	}
}

func TestAPeerThatDidNotListAllItHoldsIsAskedAboutTheDocumentsOfTheLeadingQueryWordItLeftOut(
	t *testing.T,
) {
	t.Parallel()

	documentsListedForTheLeadingQueryWord := []string{
		"https://anchored.example/",
		"https://answered.example/",
	}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: documentsListedForTheLeadingQueryWord},
		"second": {secondWord: {"https://answered.example/", "https://anchored.example/"}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second"},
		}),
		firstWord+" "+secondWord,
		&recordedSpreads{},
	)

	if len(network.crossCheckedDocumentsAsks) != 1 ||
		network.crossCheckedDocumentsAsks[0].Peer.Address != "second" {
		t.Fatalf(
			"the spread put %v, want one cross-checked documents ask to the peer that did not list all it holds",
			network.crossCheckedDocumentsAsks,
		)
	}
	wanted := documentHashesOf([]string{"https://anchored.example/"})
	if got := documentsAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, wanted,
	) {
		t.Fatalf(
			"the ask named %v, want the document listed for the leading query word that the answer left out",
			got,
		)
	}
}

func TestADocumentTheSecondRoundProvesJoinsTheDocumentsOfTheFirst(t *testing.T) {
	t.Parallel()

	documentsListedForTheLeadingQueryWord := []string{
		"https://anchored.example/",
		"https://answered.example/",
	}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: documentsListedForTheLeadingQueryWord},
		"second": {secondWord: {"https://answered.example/", "https://anchored.example/"}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1}
	observer := &recordedSpreads{}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second"},
		}),
		firstWord+" "+secondWord,
		observer,
	)

	performed := observer.performed[0]
	if performed.CrossCheckedDocumentsRound.AmountOfJoinedDocumentsFoundOnlyByCrossChecking != 1 ||
		performed.CrossCheckedDocumentsRound.AmountOfJoinedDocuments != 2 {
		t.Fatalf("the spread reported %+v, want cross-checking adding one document to the join",
			performed)
	}
	if performed.MatchedAndHeldDocumentsRound.AmountOfFullyListedQueryWords != 1 {
		t.Fatalf("the spread reported %+v, want one fully listed word", performed)
	}
}

func TestTheFullyListedWordTheFewestDocumentsAreHeldForLeads(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first": {
			firstWord: {
				"https://first.example/", "https://second.example/", "https://third.example/",
			},
		},
		"second": {secondWord: {"https://first.example/", "https://second.example/"}},
	})
	network.documentsHeldByEachPeer = map[string]int{"first": 3, "second": 2}
	observer := &recordedSpreads{}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second"},
		}),
		firstWord+" "+secondWord,
		observer,
	)

	matchedAndHeldDocumentsRound := observer.performed[0].MatchedAndHeldDocumentsRound
	if matchedAndHeldDocumentsRound.LeadingQueryWordStanding != wordjoined.RarestFullyListedQueryWord ||
		matchedAndHeldDocumentsRound.AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord != 2 {
		t.Fatalf(
			"the spread reported %+v, want the fully listed word the fewest documents are held for leading",
			matchedAndHeldDocumentsRound,
		)
	}
}

func TestAFullyListedWordLeadsOverAPartlyListedWordFewerDocumentsAreCountedFor(t *testing.T) {
	t.Parallel()

	anchored := "https://anchored.example/"
	unlisted := "https://unlisted.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {anchored, unlisted, "https://other.example/"}},
		"second": {secondWord: {
			anchored, unlisted, "https://third.example/",
			"https://fourth.example/", "https://fifth.example/",
		}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"first": 1}
	network.documentsHeldByEachPeer = map[string]int{"second": 5}
	observer := &recordedSpreads{}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second"},
		}),
		firstWord+" "+secondWord,
		observer,
	)

	performed := observer.performed[0]
	if performed.MatchedAndHeldDocumentsRound.LeadingQueryWordStanding != wordjoined.MoreCommonFullyListedQueryWord ||
		performed.MatchedAndHeldDocumentsRound.AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord != 5 {
		t.Fatalf(
			"the spread reported %+v, want the fully listed word leading over the rarer partly listed word",
			performed.MatchedAndHeldDocumentsRound,
		)
	}
	if performed.CrossCheckedDocumentsRound.AmountOfJoinedDocuments != 2 {
		t.Fatalf(
			"the spread joined %d documents, want the one both words listed and the one the partly listed word left out",
			performed.CrossCheckedDocumentsRound.AmountOfJoinedDocuments,
		)
	}
}

func TestThePartlyListedWordTheFewestDocumentsAreCountedForLeadsWhenNoWordIsFullyListed(
	t *testing.T,
) {
	t.Parallel()

	anchored := "https://anchored.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {
			firstWord: {"https://second.example/", anchored, "https://third.example/"},
		},
		"second": {secondWord: {anchored, "https://second.example/"}},
		"third": {thirdWord: {
			"https://fourth.example/", anchored, "https://second.example/",
			"https://third.example/", "https://fifth.example/",
		}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"first": 1, "second": 1, "third": 1}
	observer := &recordedSpreads{}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second"},
			thirdWord:  {"third"},
		}),
		firstWord+" "+secondWord+" "+thirdWord,
		observer,
	)

	matchedAndHeldDocumentsRound := observer.performed[0].MatchedAndHeldDocumentsRound
	if matchedAndHeldDocumentsRound.LeadingQueryWordStanding != wordjoined.RarestPartlyListedQueryWord ||
		matchedAndHeldDocumentsRound.AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord != 1 {
		t.Fatalf(
			"the spread reported %+v, want the partly listed word the fewest documents are counted for leading",
			matchedAndHeldDocumentsRound,
		)
	}
	wanted := documentHashesOf([]string{anchored, anchored})
	if got := documentsAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, wanted,
	) {
		t.Fatalf(
			"the asks named %v, want the document the leading query word listed, once per other word",
			got,
		)
	}
}

func TestAPeerThatDidNotListAllItHoldsForTwoQueryWordsIsAskedOnce(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {"https://anchored.example/"}},
		"second": {
			secondWord: {"https://answered.example/", "https://anchored.example/"},
			thirdWord:  {"https://answered.example/", "https://anchored.example/"},
		},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second"},
			thirdWord:  {"second"},
		}),
		firstWord+" "+secondWord+" "+thirdWord,
		&recordedSpreads{},
	)

	if len(network.crossCheckedDocumentsAsks) != 1 {
		t.Fatalf(
			"the spread put %v, want one cross-checked documents ask to the peer that did not list all it holds for both words",
			network.crossCheckedDocumentsAsks,
		)
	}
}

func TestADocumentOneReplicaListedForAPartlyListedWordIsAskedOfNoOtherReplica(t *testing.T) {
	t.Parallel()

	listed := "https://listed.example/"
	named := "https://named.example/"
	other := "https://other.example/"
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {listed, named, other}},
		"fourth": {firstWord: {named, other}},
		"fifth":  {firstWord: {named, other}},
		"second": {secondWord: {listed, named, other}},
		"third":  {secondWord: {listed, named, other}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1, "third": 0}
	observer := &recordedSpreads{}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first", "fourth", "fifth"},
			secondWord: {"second", "third"},
		}),
		firstWord+" "+secondWord,
		observer,
	)

	if slices.Contains(
		documentsAskedToCrossCheck(
			network.crossCheckedDocumentsAsks,
		),
		documentHashesOf([]string{listed})[0],
	) {
		t.Fatalf(
			"the spread put %v, want no ask naming the document a replica already listed",
			network.crossCheckedDocumentsAsks,
		)
	}
	crossCheckedDocumentsRound := observer.performed[0].CrossCheckedDocumentsRound
	if crossCheckedDocumentsRound.AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling != 0 ||
		crossCheckedDocumentsRound.AmountOfJoinedDocuments != 3 {
		t.Fatalf(
			"the spread reported %+v, want every document listed for the leading query word joined and none past the ceiling",
			crossCheckedDocumentsRound,
		)
	}
}

func TestTwoPartlyListedReplicasOfAQueryWordAreAskedDisjointDocuments(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	documentsListedForTheLeadingQueryWord := []string{
		answered,
		"https://first.example/",
		"https://second.example/",
		"https://third.example/",
	}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: documentsListedForTheLeadingQueryWord},
		"second": {secondWord: documentsListedForTheLeadingQueryWord},
		"third":  {secondWord: documentsListedForTheLeadingQueryWord},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1, "third": 1}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second", "third"},
		}),
		firstWord+" "+secondWord,
		&recordedSpreads{},
	)

	if len(network.crossCheckedDocumentsAsks) != 2 {
		t.Fatalf(
			"the spread put %v, want a cross-checked documents ask to each partly listed replica",
			network.crossCheckedDocumentsAsks,
		)
	}
	wanted := documentsInTheirHashOrder(documentHashesOf(documentsListedForTheLeadingQueryWord[1:]))
	got := documentsInTheirHashOrder(documentsAskedToCrossCheck(network.crossCheckedDocumentsAsks))
	if !slices.Equal(got, wanted) {
		t.Fatalf(
			"the asks named %v, want %v dealt across the replicas without a repeat",
			got,
			wanted,
		)
	}
}

func TestTheDocumentsNoPartlyListedReplicaCanTakeAreCountedPastTheCrossCheckedDocumentsCeiling(
	t *testing.T,
) {
	t.Parallel()

	const documentsOneCrossCheckedDocumentsAskNames = 1

	documentsListedForTheLeadingQueryWord := []string{
		"https://first.example/",
		"https://second.example/",
		"https://third.example/",
		"https://fourth.example/",
	}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: documentsListedForTheLeadingQueryWord},
		"second": {secondWord: documentsListedForTheLeadingQueryWord},
		"third":  {secondWord: documentsListedForTheLeadingQueryWord},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 0, "third": 0}
	observer := &recordedSpreads{}

	spreadNamingCrossCheckedDocumentsForUpTo(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second", "third"},
		}),
		documentsOneCrossCheckedDocumentsAskNames,
		observer,
	)

	crossCheckedDocumentsRound := observer.performed[0].CrossCheckedDocumentsRound
	if crossCheckedDocumentsRound.AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling != 2 ||
		crossCheckedDocumentsRound.AmountOfDocumentsSentForCrossChecking != 2 {
		t.Fatalf(
			"the spread reported %+v, want the two documents the replicas took sent and the two "+
				"no replica took past the ceiling",
			crossCheckedDocumentsRound,
		)
	}
	if len(network.crossCheckedDocumentsAsks) != 2 {
		t.Fatalf(
			"the spread put %v, want a cross-checked documents ask to each partly listed replica",
			network.crossCheckedDocumentsAsks,
		)
	}
	for _, ask := range network.crossCheckedDocumentsAsks {
		if len(ask.Documents) != documentsOneCrossCheckedDocumentsAskNames {
			t.Fatalf(
				"the ask to peer %q named %v, want the one document the ceiling allows",
				ask.Peer.Address,
				ask.Documents,
			)
		}
	}
}

func TestADocumentSentToCrossCheckForTwoQueryWordsIsCountedForEach(t *testing.T) {
	t.Parallel()

	documentsListedForTheLeadingQueryWord := []string{
		"https://first.example/",
		"https://second.example/",
	}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: documentsListedForTheLeadingQueryWord},
		"second": {secondWord: documentsListedForTheLeadingQueryWord},
		"third":  {thirdWord: documentsListedForTheLeadingQueryWord},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 0, "third": 0}
	observer := &recordedSpreads{}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second"},
			thirdWord:  {"third"},
		}),
		firstWord+" "+secondWord+" "+thirdWord,
		observer,
	)

	crossCheckedDocumentsRound := observer.performed[0].CrossCheckedDocumentsRound
	if crossCheckedDocumentsRound.AmountOfDocumentsSentForCrossChecking != 4 ||
		crossCheckedDocumentsRound.AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling != 0 {
		t.Fatalf(
			"the spread reported %+v, want both documents sent once for each of the two other "+
				"query words",
			crossCheckedDocumentsRound,
		)
	}
}

func TestAPeerThatHoldsNoneOfTheNamedDocumentsLeavesTheJoinOfTheFirstRoundWhole(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {"https://answered.example/", "https://anchored.example/"}},
		"second": {
			secondWord: {"https://answered.example/", "https://anchored.example/"},
		},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1}
	network.peersHoldingNoNamedDocument = map[string]struct{}{"second": {}}
	observer := &recordedSpreads{}

	spreadOfTheQuery(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second"},
		}),
		firstWord+" "+secondWord,
		observer,
	)

	wanted := documentHashesOf([]string{"https://answered.example/"})
	if got := distinctDocumentsAskedMetadataFor(network.urlMetadataAsks); !slices.Equal(
		got, wanted,
	) {
		t.Fatalf("the join kept %v, want the document the first round already proved", got)
	}
	if observer.performed[0].CrossCheckedDocumentsRound.AmountOfEmptyCrossCheckedDocumentsAnswers != 1 {
		t.Fatalf(
			"the spread reported %d empty answers, want the one the peer sent",
			observer.performed[0].CrossCheckedDocumentsRound.AmountOfEmptyCrossCheckedDocumentsAnswers,
		)
	}
}

func TestAQueryWhoseWordsAreAllFullyListedCrossChecksNoDocument(t *testing.T) {
	t.Parallel()

	shared := []string{"https://shared.example/"}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: shared},
		"second": {secondWord: shared},
	})
	observer := &recordedSpreads{}

	spreadOf(network, observer)

	if len(network.crossCheckedDocumentsAsks) != 0 || len(network.urlMetadataAsks) == 0 {
		t.Fatalf(
			"the spread put %d cross-checked documents asks and %d metadata asks, want none and some",
			len(network.crossCheckedDocumentsAsks),
			len(network.urlMetadataAsks),
		)
	}
	if observer.performed[0].MatchedAndHeldDocumentsRound.AmountOfFullyListedQueryWords != 2 {
		t.Fatalf(
			"the spread reported %d fully listed query words, want both",
			observer.performed[0].MatchedAndHeldDocumentsRound.AmountOfFullyListedQueryWords,
		)
	}
}

func TestAPeerThatAnsweredNothingInTheFirstRoundIsAskedInTheSecond(t *testing.T) {
	t.Parallel()

	shared := []string{"https://shared.example/"}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: shared},
		"second": {secondWord: shared},
	})
	network.silentPeers["second"] = struct{}{}

	spreadChoosing(network, peersOfEachQueryWord(map[string][]string{
		firstWord:  {"first"},
		secondWord: {"second"},
	}), &recordedSpreads{})

	if len(network.crossCheckedDocumentsAsks) != 1 ||
		network.crossCheckedDocumentsAsks[0].Peer.Address != "second" {
		t.Fatalf(
			"the spread put %v, want one cross-checked documents ask to the peer that stayed silent",
			network.crossCheckedDocumentsAsks,
		)
	}
	wanted := documentHashesOf(shared)
	if got := distinctDocumentsAskedMetadataFor(network.urlMetadataAsks); !slices.Equal(
		got, wanted,
	) {
		t.Fatalf("the join kept %v, want the document the silent peer answered for", got)
	}
}

func TestTheFirstRoundKeepsOnlyAThirdOfTheTimeTheQueryHasLeft(t *testing.T) {
	t.Parallel()

	const queryBudget = 3 * time.Second

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://shared.example/"}},
		"second": {secondWord: {"https://shared.example/"}},
	})

	spreadWithin(network, queryBudget)

	if len(network.timeLeftInEachRoundInTheirOrder) != 3 {
		t.Fatalf(
			"the spread ran %d rounds of peer calls, want three",
			len(network.timeLeftInEachRoundInTheirOrder),
		)
	}
	timeLeftInTheFirstRound := network.timeLeftInEachRoundInTheirOrder[0]
	if timeLeftInTheFirstRound < queryBudget/3-queryBudget/10 ||
		timeLeftInTheFirstRound > queryBudget/3+queryBudget/10 {
		t.Fatalf(
			"the first round kept %s of the %s the query has, want a third",
			timeLeftInTheFirstRound,
			queryBudget,
		)
	}
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

	foundDocuments := spreadOf(network, &recordedSpreads{}).FoundDocuments

	if len(network.urlMetadataAsks) != 0 || len(foundDocuments) != 0 {
		t.Fatalf(
			"asked %d peers and found %d documents, want none of either",
			len(network.urlMetadataAsks),
			len(foundDocuments),
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

	for _, ask := range network.matchedAndHeldDocumentsAsks {
		if ask.Peer.Address == "first" && ask.Word != yacymodel.WordHash(firstWord) {
			t.Fatalf("peer %q was asked what it holds for a word it is not responsible for",
				ask.Peer.Address)
		}
		if ask.Peer.Address == "second" && ask.Word != yacymodel.WordHash(secondWord) {
			t.Fatalf("peer %q was asked what it holds for a word it is not responsible for",
				ask.Peer.Address)
		}
	}
	if len(network.matchedAndHeldDocumentsAsks) != 2 {
		t.Fatalf(
			"%d matched and cross-checked documents asks were put, want one for each responsible peer",
			len(network.matchedAndHeldDocumentsAsks),
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
	if performed.MatchedAndHeldDocumentsRound.AmountOfQueryWords != 2 ||
		performed.CrossCheckedDocumentsRound.AmountOfJoinedDocuments != 1 {
		t.Fatalf("the spread reported %+v, want two words and one joined document",
			performed)
	}
	if performed.MatchedAndHeldDocumentsRound.AmountOfQueryWordsHeldByNoPeer != 0 ||
		performed.URLMetadataRound.AmountOfLookedUpDocumentsWithMetadata != 1 {
		t.Fatalf(
			"the spread reported %+v, want every word held and the joined document back",
			performed,
		)
	}
}

func TestAQueryWordHeldByNoPeerIsReported(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://only-first.example/"}},
		"second": {firstWord: {"https://only-first.example/"}},
	})
	observer := &recordedSpreads{}

	spreadOf(network, observer)

	if observer.performed[0].MatchedAndHeldDocumentsRound.AmountOfQueryWordsHeldByNoPeer != 1 {
		t.Fatalf(
			"the spread reported %d query words held by no peer, want the one word nobody held",
			observer.performed[0].MatchedAndHeldDocumentsRound.AmountOfQueryWordsHeldByNoPeer,
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
	if performed.CrossCheckedDocumentsRound.AmountOfJoinedDocuments != 2 ||
		performed.URLMetadataRound.AmountOfLookedUpDocuments != 1 {
		t.Fatalf(
			"the spread reported %+v, want two joined documents and one asked metadata for",
			performed,
		)
	}
	if performed.URLMetadataRound.AmountOfLookedUpDocumentsWithMetadata != 1 {
		t.Fatalf(
			"the spread reported %d documents back, want the one it asked metadata for",
			performed.URLMetadataRound.AmountOfLookedUpDocumentsWithMetadata,
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
		spreadChoosingPeersBy(
			responsiblePeers{},
			wordjoined.New(
				network,
				network,
				metadataDocumentsCeiling,
				asksForCrossCheckedDocuments,
				crossCheckedDocumentsCeiling,
				peerItemsCeiling,
				onePartitionOfTheRing,
				peersOfOneWord,
				&recordedSpreads{},
			),
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

func TestOnlyTheJoinedDocumentsAPeerAnsweredAreFound(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	})
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"first": {firstWord: {answered, "https://unjoined.example/"}},
	}

	foundDocuments := spreadOf(network, &recordedSpreads{}).FoundDocuments

	wanted := documentHashOf(t, answered)
	documents := map[yacymodel.URLHash]struct{}{}
	for _, foundDocument := range foundDocuments {
		documents[foundDocument.Hash] = struct{}{}
	}
	if _, cameBack := documents[wanted]; !cameBack || len(documents) != 1 {
		t.Fatalf("the spread answered %v, want only the joined document the peer answered",
			documents)
	}
}

func TestTheAnswersCarryTheWordsOfTheQuery(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	})

	answers := spreadOf(network, &recordedSpreads{})

	want := []yacymodel.Hash{yacymodel.WordHash(firstWord), yacymodel.WordHash(secondWord)}
	if !slices.Equal(answers.QueryWords, want) {
		t.Fatalf("the answers carry the query words %v, want %v", answers.QueryWords, want)
	}
}

func TestAFoundDocumentIsCountedForTheWordThePeerWasAskedAbout(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	})
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"first": {firstWord: {answered}},
	}
	network.countsAWordWithEachItem = true

	foundDocuments := spreadOf(network, &recordedSpreads{}).FoundDocuments

	if len(foundDocuments) != 1 {
		t.Fatalf("the spread found %v, want the one document the peer answered", foundDocuments)
	}
	hitsPerQueryWord := foundDocuments[0].HitsPerQueryWord
	_, secondWordHasHits := hitsPerQueryWord[yacymodel.WordHash(secondWord)]
	if hitsPerQueryWord[yacymodel.WordHash(firstWord)] != 3 || secondWordHasHits {
		t.Fatalf("the found document holds the hits %v, want the hits of the word the ask named",
			hitsPerQueryWord)
	}
}

func TestTheCountsOfEachQueryWordComeTogetherOnTheJoinedDocument(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	})
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	}
	network.countsAWordWithEachItem = true

	foundDocuments := spreadOf(network, &recordedSpreads{}).FoundDocuments

	if len(foundDocuments) != 1 {
		t.Fatalf("the spread found %v, want the joined document once", foundDocuments)
	}
	hitsPerQueryWord := foundDocuments[0].HitsPerQueryWord
	if hitsPerQueryWord[yacymodel.WordHash(firstWord)] != 3 ||
		hitsPerQueryWord[yacymodel.WordHash(secondWord)] != 3 {
		t.Fatalf("the found document holds the hits %v, want the hits of each query word",
			hitsPerQueryWord)
	}
}

func TestAJoinedDocumentNoPeerAnsweredIsFoundThroughItsMetadata(t *testing.T) {
	t.Parallel()

	joined := "https://joined.example/"
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {joined}, secondWord: {joined}},
		"second": {firstWord: {joined}, secondWord: {joined}},
	})

	foundDocuments := spreadOf(network, &recordedSpreads{}).FoundDocuments

	if len(foundDocuments) != 1 || foundDocuments[0].Hash != documentHashOf(t, joined) {
		t.Fatalf("the spread found %v, want the joined document once", foundDocuments)
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

	want := documentsHeldForBothQueryWords(512)
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

	want := documentsHeldForBothQueryWords(512)
	if got := answers.DocumentsHeldPerQueryWord; !maps.Equal(got, want) {
		t.Fatalf("the answers carry %v documents per query word, want %v", got, want)
	}
}

func TestEveryPartitionOfAQueryWordAddsWhatItsReplicasCounted(t *testing.T) {
	t.Parallel()

	got := documentsHeldPerQueryWordAcrossPartitions(
		map[string]int{"first": 100, "second": 100, "third": 100, "fourth": 10},
		map[string]struct{}{},
		2,
		responsiblePeers{partitionOfEachPeer: map[string]uint{"fourth": 1}},
	)

	if want := documentsHeldForBothQueryWords(110); !maps.Equal(got, want) {
		t.Fatalf("the answers carry %v documents per query word, want %v", got, want)
	}
}

func TestAPartitionOfAnEvenAmountOfCountsTakesTheLowerMiddleOne(t *testing.T) {
	t.Parallel()

	got := documentsHeldPerQueryWordAcrossPartitions(
		map[string]int{"first": 10, "second": 20},
		map[string]struct{}{},
		1,
		responsiblePeers{},
	)

	if want := documentsHeldForBothQueryWords(10); !maps.Equal(got, want) {
		t.Fatalf("the answers carry %v documents per query word, want %v", got, want)
	}
}

func TestAPartitionNoPeerCountedTakesTheMiddleOfThePartitionsThatWereCounted(t *testing.T) {
	t.Parallel()

	got := documentsHeldPerQueryWordAcrossPartitions(
		map[string]int{"first": 10, "second": 30},
		map[string]struct{}{"third": {}},
		3,
		responsiblePeers{partitionOfEachPeer: map[string]uint{"second": 1, "third": 2}},
	)

	if want := documentsHeldForBothQueryWords(10 + 30 + 10); !maps.Equal(got, want) {
		t.Fatalf("the answers carry %v documents per query word, want %v", got, want)
	}
}

func TestAQueryWordNoPeerCountedCarriesNoDocumentsHeld(t *testing.T) {
	t.Parallel()

	got := documentsHeldPerQueryWordAcrossPartitions(
		map[string]int{},
		map[string]struct{}{"first": {}, "second": {}},
		2,
		responsiblePeers{partitionOfEachPeer: map[string]uint{"second": 1}},
	)

	if len(got) != 0 {
		t.Fatalf("the answers carry %v documents per query word, want none", got)
	}
}

func documentsHeldPerQueryWordAcrossPartitions(
	documentsHeldByEachPeer map[string]int,
	peersCountingNoDocument map[string]struct{},
	partitions yacymodel.DHTRingPartitions,
	choice responsiblePeers,
) map[yacymodel.Hash]int {
	network := networkOf(map[string]map[string][]string{})
	network.documentsHeldByEachPeer = documentsHeldByEachPeer
	network.peersCountingNoDocument = peersCountingNoDocument

	return spreadOverPeers(
		spreadChoosingPeersBy(
			choice,
			wordjoined.New(
				network,
				network,
				metadataDocumentsCeiling,
				asksForCrossCheckedDocuments,
				crossCheckedDocumentsCeiling,
				peerItemsCeiling,
				partitions,
				peersHoldingOneWord,
				&recordedSpreads{},
			),
		),
		peersAt(addressesAcross(documentsHeldByEachPeer, peersCountingNoDocument)),
	).DocumentsHeldPerQueryWord
}

func addressesAcross(
	documentsHeldByEachPeer map[string]int,
	peersCountingNoDocument map[string]struct{},
) []string {
	addresses := make([]string, 0, len(documentsHeldByEachPeer)+len(peersCountingNoDocument))
	for address := range documentsHeldByEachPeer {
		addresses = append(addresses, address)
	}
	for address := range peersCountingNoDocument {
		addresses = append(addresses, address)
	}
	slices.Sort(addresses)

	return addresses
}

func documentsHeldForBothQueryWords(documentsHeld int) map[yacymodel.Hash]int {
	return map[yacymodel.Hash]int{
		yacymodel.WordHash(firstWord):  documentsHeld,
		yacymodel.WordHash(secondWord): documentsHeld,
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
	matchedAndHeldDocumentsRound := performed.MatchedAndHeldDocumentsRound
	if performed.URLMetadataRound.AmountOfJoinedDocumentsWithMetadata != 1 ||
		matchedAndHeldDocumentsRound.AmountOfMatchedDocumentsAcrossAnswers != 2 ||
		matchedAndHeldDocumentsRound.AmountOfMatchedDocumentsWithAPosting != 2 {
		t.Fatalf(
			"the spread reported %+v, want the joined document answered once and two counts",
			performed,
		)
	}
	if !slices.Equal(
		matchedAndHeldDocumentsRound.AmountOfDocumentsHeldInEachAnswer, []int{512, 512},
	) {
		t.Fatalf(
			"the spread reported %v documents held per query word, want 512 for each answer",
			matchedAndHeldDocumentsRound.AmountOfDocumentsHeldInEachAnswer,
		)
	}
}

func TestTheAsksOfAQueryWordNameItsReplicasInChoiceOrderWithTheirPartitions(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{})

	spreadAcrossPartitions(network, responsiblePeers{
		peerAddressesPerWord: map[string][]string{
			firstWord:  {"nearest-of-first", "next-of-first"},
			secondWord: {"nearest-of-second"},
		},
		partitionOfEachPeer: map[string]uint{
			"nearest-of-first":  1,
			"next-of-first":     0,
			"nearest-of-second": 1,
		},
	}, twoPartitionsOfTheRing)

	wanted := []string{"nearest-of-first in partition 1", "next-of-first in partition 0"}
	if got := replicasAskedForTheWordInOrder(
		yacymodel.WordHash(firstWord), network.matchedAndHeldDocumentsAsks,
	); !slices.Equal(got, wanted) {
		t.Fatalf("the spread asked %v for the word, want %v", got, wanted)
	}
}

func spreadAcrossPartitions(
	network *peerNetwork,
	choice responsiblePeers,
	partitions yacymodel.DHTRingPartitions,
) {
	spreadOverPeers(
		spreadChoosingPeersBy(
			choice,
			wordjoined.New(
				network,
				network,
				metadataDocumentsCeiling,
				asksForCrossCheckedDocuments,
				crossCheckedDocumentsCeiling,
				peerItemsCeiling,
				partitions,
				peersHoldingOneWord,
				&recordedSpreads{},
			),
		),
		peersAt([]string{"nearest-of-first", "next-of-first", "nearest-of-second"}),
	)
}

func replicasAskedForTheWordInOrder(
	word yacymodel.Hash,
	asks []peerasks.MatchedAndHeldDocumentsAsk,
) []string {
	var replicasAsked []string
	for _, ask := range asks {
		if ask.Word != word {
			continue
		}
		replicasAsked = append(
			replicasAsked,
			fmt.Sprintf("%s in partition %d", ask.Peer.Address, ask.Partition),
		)
	}

	return replicasAsked
}

type spreadChoosingPeers struct {
	choice responsiblePeers
	spread wordjoined.Spread
}

func spreadChoosingPeersBy(choice responsiblePeers, spread wordjoined.Spread) spreadChoosingPeers {
	return spreadChoosingPeers{choice: choice, spread: spread}
}

func (s spreadChoosingPeers) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) queryanswers.AnsweredQuery {
	return s.spread.SpreadOverPeers(
		ctx,
		query,
		s.choice.ChosenPeersPerQueryWordFor(ctx, query.TermHashes(), askablePeers),
	)
}
