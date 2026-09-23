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
	peerjudgementledgersmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementledgers/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord                       = "berlin"
	secondWord                      = "weather"
	thirdWord                       = "rain"
	urlMetadataAskDocumentsCeiling  = 10
	documentsOneURLMetadataAskNames = 1
	crossCheckedDocumentsCeiling    = 10
	peerItemsCeiling                = 10
	peersHoldingOneWord             = 24
	onePartitionOfTheRing           = 1
	compoundWordsCeiling            = 4
	twoPartitionsOfTheRing          = 2
	judgementLedgerCapacity         = 16
	crossCheckRetrialInterval       = 24 * time.Hour
)

var (
	versionOfThePeer      = yacymodel.Some(yacymodel.SoftwareVersion{Release: 1.925})
	versionAfterAnUpgrade = yacymodel.Some(yacymodel.SoftwareVersion{Release: 1.93})
)

type peerNetwork struct {
	documentsPerWordPerPeer                         map[string]map[string][]string
	answeredItemsPerWordPerPeer                     map[string]map[string][]string
	countsAWordWithEachItem                         bool
	documentsHeldForEveryWord                       int
	documentsHeldByEachPeer                         map[string]int
	documentsPerAnswerOfEachPeer                    map[string]int
	peersCountingNoDocument                         map[string]struct{}
	replicasPutPerWord                              map[string]int
	matchedAndHeldDocumentsAsks                     []peerasks.SearchDocumentsAsk
	matchedAndHeldDocumentsAsksPut                  []peerasks.SearchDocumentsAsk
	crossCheckedDocumentsAsks                       []peerasks.SearchDocumentsAsk
	urlMetadataAsks                                 []peerasks.URLMetadataAsk
	silentPeers                                     map[string]struct{}
	peersSilentInTheCrossCheckedDocuments           map[string]struct{}
	peersHoldingNoDocumentToMatch                   map[string]struct{}
	peersListingDocumentsOutsideTheDocumentsToMatch map[string]struct{}
	timeLeftInEachRoundInTheirOrder                 []time.Duration
}

func networkOf(documentsPerWordPerPeer map[string]map[string][]string) *peerNetwork {
	return &peerNetwork{
		documentsPerWordPerPeer:                         documentsPerWordPerPeer,
		peersCountingNoDocument:                         map[string]struct{}{},
		silentPeers:                                     map[string]struct{}{},
		peersSilentInTheCrossCheckedDocuments:           map[string]struct{}{},
		peersHoldingNoDocumentToMatch:                   map[string]struct{}{},
		peersListingDocumentsOutsideTheDocumentsToMatch: map[string]struct{}{},
	}
}

func (n *peerNetwork) AskForSearchDocuments(
	ctx context.Context,
	asks []peerasks.SearchDocumentsAsk,
) peerasks.SearchDocumentsAskOutcomes {
	if len(asks) > 0 && len(asks[0].DocumentsToMatch) > 0 {
		return n.crossCheckedDocumentsAskOutcomes(ctx, asks)
	}
	n.matchedAndHeldDocumentsAsks = append(n.matchedAndHeldDocumentsAsks, asks...)
	n.recordTimeLeftIn(ctx)

	askOutcomes := make(peerasks.SearchDocumentsAskOutcomes, 0, len(asks))
	for _, ask := range asks {
		askOutcomes = append(askOutcomes, peerasks.SearchDocumentsAskOutcome{Ask: ask})
	}
	placesPut := n.placesOfTheMatchedAndHeldDocumentsAsksPutAmong(asks)
	for place, ask := range asks {
		if _, put := placesPut[place]; !put {
			continue
		}
		n.matchedAndHeldDocumentsAsksPut = append(n.matchedAndHeldDocumentsAsksPut, ask)
		askOutcomes[place].Put = true
		if _, silent := n.silentPeers[ask.Peer.Address]; silent {
			continue
		}
		askOutcomes[place].Answer = yacymodel.Some(peerasks.AnsweredSearchDocumentsAsk{
			Ask:                       ask,
			DocumentsListedForTheWord: n.documentsListedBy(ask.Peer.Address, ask.Word),
			MatchedDocuments: n.matchedDocumentsOf(
				documentsPerWordOf(n.answeredItemsPerWordPerPeer, ask.Peer.Address, ask.Word),
			),
			AmountOfDocumentsHeldForTheWord: n.documentsCountedBy(ask.Peer.Address, ask.Word),
		})
	}

	return askOutcomes
}

type wordInPartition struct {
	word      yacymodel.Hash
	partition uint
}

func (n *peerNetwork) placesOfTheMatchedAndHeldDocumentsAsksPutAmong(
	asks []peerasks.SearchDocumentsAsk,
) map[int]struct{} {
	asksPutPerWordInPartition := map[wordInPartition]int{}
	placesPut := make(map[int]struct{}, len(asks))
	for place, ask := range asks {
		key := wordInPartition{word: ask.Word, partition: ask.Partition}
		replicasPut, limited := n.replicasPutFor(ask.Word)
		if limited && asksPutPerWordInPartition[key] == replicasPut {
			continue
		}
		asksPutPerWordInPartition[key]++
		placesPut[place] = struct{}{}
	}

	return placesPut
}

func (n *peerNetwork) replicasPutFor(word yacymodel.Hash) (int, bool) {
	for spelledWord, replicasPut := range n.replicasPutPerWord {
		if yacymodel.WordHash(spelledWord) == word {
			return replicasPut, true
		}
	}

	return 0, false
}

func (n *peerNetwork) crossCheckedDocumentsAskOutcomes(
	ctx context.Context,
	asks []peerasks.SearchDocumentsAsk,
) peerasks.SearchDocumentsAskOutcomes {
	n.crossCheckedDocumentsAsks = append(n.crossCheckedDocumentsAsks, asks...)
	n.recordTimeLeftIn(ctx)

	askOutcomes := make(peerasks.SearchDocumentsAskOutcomes, 0, len(asks))
	for _, ask := range asks {
		askOutcome := peerasks.SearchDocumentsAskOutcome{Ask: ask, Put: true}
		if _, silent := n.peersSilentInTheCrossCheckedDocuments[ask.Peer.Address]; !silent {
			documentsToMatchHeld := n.documentsToMatchHeldBy(ask)
			askOutcome.Answer = yacymodel.Some(peerasks.AnsweredSearchDocumentsAsk{
				Ask:                       ask,
				DocumentsListedForTheWord: documentsToMatchHeld,
				MatchedDocuments:          n.matchedDocumentsOf(documentsToMatchHeld),
			})
		}
		askOutcomes = append(askOutcomes, askOutcome)
	}

	return askOutcomes
}

func (n *peerNetwork) documentsToMatchHeldBy(
	ask peerasks.SearchDocumentsAsk,
) []yacymodel.URLHash {
	if _, holdsNothing := n.peersHoldingNoDocumentToMatch[ask.Peer.Address]; holdsNothing {
		return nil
	}

	heldDocuments := documentsPerWordOf(n.documentsPerWordPerPeer, ask.Peer.Address, ask.Word)
	if _, listsMore := n.peersListingDocumentsOutsideTheDocumentsToMatch[ask.Peer.Address]; listsMore {
		return heldDocuments
	}

	documentsToMatchHeld := make([]yacymodel.URLHash, 0, len(ask.DocumentsToMatch))
	for _, document := range ask.DocumentsToMatch {
		if !slices.Contains(heldDocuments, document) {
			continue
		}
		documentsToMatchHeld = append(documentsToMatchHeld, document)
	}

	return documentsToMatchHeld
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
	versionOfEachPeer    map[string]yacymodel.Optional[yacymodel.SoftwareVersion]
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
		peer.Version = r.versionOfEachPeer[peer.Address]
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

func answeredQueryFrom(
	network *peerNetwork,
	observer wordjoined.WordJoinedSpreadObserver,
) queryanswers.AnsweredQuery {
	return answeredQueryWith(responsiblePeers{}, network, observer)
}

func answeredQueryWith(
	choice responsiblePeers,
	network *peerNetwork,
	observer wordjoined.WordJoinedSpreadObserver,
) queryanswers.AnsweredQuery {
	return answeredQueryUnder(
		urlMetadataAskDocumentsCeiling,
		network,
		choice,
		observer,
	)
}

func answeredQueryUnder(
	urlMetadataAskDocumentsCeiling int,
	network *peerNetwork,
	choice responsiblePeers,
	observer wordjoined.WordJoinedSpreadObserver,
) queryanswers.AnsweredQuery {
	return spreadOverPeers(
		newSpreadOverChosenPeers(
			choice,
			wordjoined.New(
				network,
				network,
				judgementsOfTheCrossCheck(),
				urlMetadataAskDocumentsCeiling,
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
	spread spreadOverChosenPeers,
	askablePeers []peerdirectory.AskablePeer,
) queryanswers.AnsweredQuery {
	return spread.SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom(firstWord+" "+secondWord, ""),
		askablePeers,
	)
}

func spreadUnder(
	crossCheckedDocumentsCeiling int,
	network *peerNetwork,
	choice responsiblePeers,
	observer wordjoined.WordJoinedSpreadObserver,
) {
	spreadOverPeers(
		newSpreadOverChosenPeers(
			choice,
			wordjoined.New(
				network,
				network,
				judgementsOfTheCrossCheck(),
				urlMetadataAskDocumentsCeiling,
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

func spreadTheQuery(
	query string,
	network *peerNetwork,
	choice responsiblePeers,
	observer wordjoined.WordJoinedSpreadObserver,
) {
	newSpreadOverChosenPeers(
		choice,
		wordjoined.New(
			network,
			network,
			judgementsOfTheCrossCheck(),
			urlMetadataAskDocumentsCeiling,
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

func spreadWithin(queryBudget time.Duration, network *peerNetwork) {
	ctx, endQuery := context.WithTimeout(context.Background(), queryBudget)
	defer endQuery()

	newSpreadOverChosenPeers(
		responsiblePeers{},
		wordjoined.New(
			network,
			network,
			judgementsOfTheCrossCheck(),
			urlMetadataAskDocumentsCeiling,
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

func documentsAskedToCrossCheck(asks []peerasks.SearchDocumentsAsk) []yacymodel.URLHash {
	documents := make([]yacymodel.URLHash, 0, len(asks))
	for _, ask := range asks {
		documents = append(documents, ask.DocumentsToMatch...)
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

func judgementsOfTheCrossCheck() peerjudgements.Judgements {
	return peerjudgements.New(
		wordjoined.ListsOnlyTheCrossCheckedDocuments,
		peerjudgementledgersmemory.New(judgementLedgerCapacity),
		crossCheckRetrialInterval,
		time.Now,
	)
}

func judgementsWhere(judgedPeers ...peerjudgements.JudgedPeer) peerjudgements.Judgements {
	judgements := judgementsOfTheCrossCheck()
	judgements.Add(context.Background(), judgedPeers)

	return judgements
}

func peerJudged(address string, judgement peerjudgements.Judgement) peerjudgements.JudgedPeer {
	return peerjudgements.JudgedPeerFrom(
		peerAt(address).Hash, yacymodel.None[yacymodel.SoftwareVersion](), judgement,
	)
}

func spreadJudgingThePeers(
	network *peerNetwork,
	choice responsiblePeers,
	judgements wordjoined.PeerJudgements,
	observer wordjoined.WordJoinedSpreadObserver,
) {
	newSpreadOverChosenPeers(
		choice,
		wordjoined.New(
			network,
			network,
			judgements,
			urlMetadataAskDocumentsCeiling,
			crossCheckedDocumentsCeiling,
			peerItemsCeiling,
			onePartitionOfTheRing,
			peersHoldingOneWord,
			observer,
		),
	).SpreadOverPeers(
		context.Background(),
		searchquery.QueryFrom(firstWord+" "+secondWord, ""),
		peersAt([]string{"first", "second"}),
	)
}

func networkOfAReplicaTheFirstRoundLeavesUnasked() *peerNetwork {
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://anchored.example/", "https://answered.example/"}},
		"second": {secondWord: {"https://answered.example/", "https://anchored.example/"}},
		"third":  {secondWord: {"https://answered.example/", "https://anchored.example/"}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1, "third": 1}
	network.replicasPutPerWord = map[string]int{secondWord: 1}

	return network
}

func replicasOfTheTwoQueryWords() responsiblePeers {
	replicas := peersOfEachQueryWord(map[string][]string{
		firstWord:  {"first"},
		secondWord: {"second", "third"},
	})
	replicas.versionOfEachPeer = map[string]yacymodel.Optional[yacymodel.SoftwareVersion]{
		"third": versionOfThePeer,
	}

	return replicas
}

func TestAPeerThatClaimsAnotherVersionIsAskedAgain(t *testing.T) {
	t.Parallel()

	network := networkOfAReplicaTheFirstRoundLeavesUnasked()
	network.peersListingDocumentsOutsideTheDocumentsToMatch = map[string]struct{}{"third": {}}
	judgements := judgementsOfTheCrossCheck()
	replicas := replicasOfTheTwoQueryWords()

	spreadJudgingThePeers(network, replicas, judgements, &recordedSpreads{})
	replicas.versionOfEachPeer = map[string]yacymodel.Optional[yacymodel.SoftwareVersion]{
		"third": versionAfterAnUpgrade,
	}
	spreadJudgingThePeers(network, replicas, judgements, &recordedSpreads{})

	if amount := amountOfCrossChecksPutTo("third", network.crossCheckedDocumentsAsks); amount != 2 {
		t.Fatalf(
			"the spreads asked the peer to cross-check %d times, want again at the version it claims now",
			amount,
		)
	}
}

func addressesAskedToCrossCheck(asks []peerasks.SearchDocumentsAsk) []string {
	addresses := make([]string, 0, len(asks))
	for _, ask := range asks {
		addresses = append(addresses, ask.Peer.Address)
	}

	return addresses
}

func amountOfCrossChecksPutTo(address string, asks []peerasks.SearchDocumentsAsk) int {
	amount := 0
	for _, ask := range asks {
		if ask.Peer.Address == address {
			amount++
		}
	}

	return amount
}

func TestAReplicaTheFirstRoundLeftUnaskedIsAskedAboutTheDocumentsItsPartitionLeftOut(
	t *testing.T,
) {
	t.Parallel()

	network := networkOfAReplicaTheFirstRoundLeavesUnasked()

	spreadTheQuery(
		firstWord+" "+secondWord, network, replicasOfTheTwoQueryWords(), &recordedSpreads{},
	)

	if got := addressesAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, []string{"third"},
	) {
		t.Fatalf(
			"the spread asked %v to cross-check, want only the replica the first round left unasked",
			got,
		)
	}
	ask := network.crossCheckedDocumentsAsks[0]
	wanted := documentHashesOf([]string{"https://anchored.example/"})
	if !slices.Equal(ask.DocumentsToMatch, wanted) || ask.ItemsCeiling != peerItemsCeiling ||
		ask.Partition != 0 {
		t.Fatalf(
			"the ask reads %+v, want the document the partition left out, the partition and "+
				"the items ceiling of a peer",
			ask,
		)
	}
}

func TestEachPeerGetsAtMostOneSearchCallPerQuery(t *testing.T) {
	t.Parallel()

	both := []string{"https://anchored.example/", "https://answered.example/"}
	reversed := []string{both[1], both[0]}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: both},
		"second": {secondWord: reversed},
		"third":  {secondWord: both, thirdWord: both},
		"fourth": {secondWord: both},
		"fifth":  {thirdWord: reversed},
		"sixth":  {thirdWord: both},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1, "fifth": 1}
	network.replicasPutPerWord = map[string]int{secondWord: 1, thirdWord: 1}

	spreadTheQuery(
		firstWord+" "+secondWord+" "+thirdWord,
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second", "third", "fourth"},
			thirdWord:  {"fifth", "third", "sixth"},
		}),
		&recordedSpreads{},
	)

	searchCallsPerPeer := map[string]int{}
	for _, ask := range network.matchedAndHeldDocumentsAsksPut {
		searchCallsPerPeer[ask.Peer.Address]++
	}
	for _, ask := range network.crossCheckedDocumentsAsks {
		searchCallsPerPeer[ask.Peer.Address]++
	}
	for address, searchCalls := range searchCallsPerPeer {
		if searchCalls > 1 {
			t.Fatalf("peer %q got %d search calls, want at most one", address, searchCalls)
		}
	}
	if got := addressesAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, []string{"third", "fourth", "sixth"},
	) {
		t.Fatalf("the spread asked %v to cross-check, want each unasked replica once", got)
	}
}

func TestTheStandingsOfEveryChosenPeerAreLookedUpOnceBeforeTheFirstRound(t *testing.T) {
	t.Parallel()

	judgements := &recordedJudgements{PeerJudgements: judgementsOfTheCrossCheck()}

	spreadJudgingThePeers(
		networkOfAReplicaTheFirstRoundLeavesUnasked(),
		replicasOfTheTwoQueryWords(),
		judgements,
		&recordedSpreads{},
	)

	wanted := []peerjudgements.PeerAtVersion{
		{Peer: peerAt("first").Hash},
		{Peer: peerAt("second").Hash},
		{Peer: peerAt("third").Hash, Version: versionOfThePeer},
	}
	if judgements.amountOfLookups != 1 || !slices.Equal(judgements.peersLookedUp, wanted) {
		t.Fatalf(
			"the spread looked up %v in %d lookups, want every chosen peer at the version it claims in one",
			judgements.peersLookedUp,
			judgements.amountOfLookups,
		)
	}
}

func TestIgnoringPeersAreAskedFirstAndHonoringPeersLastInTheFirstRound(t *testing.T) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{})

	spreadJudgingThePeers(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"honoring", "second", "ignoring"},
		}),
		judgementsWhere(
			peerJudged("honoring", peerjudgements.Honored),
			peerJudged("ignoring", peerjudgements.Ignored),
		),
		&recordedSpreads{},
	)

	wanted := []string{
		"ignoring in partition 0", "second in partition 0", "honoring in partition 0",
	}
	if got := replicasAskedForTheWordInOrder(
		yacymodel.WordHash(secondWord), network.matchedAndHeldDocumentsAsks,
	); !slices.Equal(got, wanted) {
		t.Fatalf("the first round asked %v for the word, want %v", got, wanted)
	}
}

func TestHonoringPeersAreAskedFirstInTheSecondRound(t *testing.T) {
	t.Parallel()

	both := []string{"https://answered.example/", "https://anchored.example/"}
	network := networkOf(map[string]map[string][]string{
		"first":    {firstWord: both},
		"second":   {secondWord: both},
		"plain":    {secondWord: both},
		"honoring": {secondWord: both},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1}
	network.replicasPutPerWord = map[string]int{secondWord: 1}

	spreadJudgingThePeers(
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second", "plain", "honoring"},
		}),
		judgementsWhere(peerJudged("honoring", peerjudgements.Honored)),
		&recordedSpreads{},
	)

	if got := addressesAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, []string{"honoring", "plain"},
	) {
		t.Fatalf("the second round asked %v, want the honoring peer first", got)
	}
}

func TestAPeerThatListedADocumentOutsideTheDocumentsToMatchIsNotAskedToCrossCheckByTheNextQuery(
	t *testing.T,
) {
	t.Parallel()

	network := networkOfAReplicaTheFirstRoundLeavesUnasked()
	network.peersListingDocumentsOutsideTheDocumentsToMatch = map[string]struct{}{"third": {}}
	judgements := judgementsOfTheCrossCheck()

	spreadJudgingThePeers(network, replicasOfTheTwoQueryWords(), judgements, &recordedSpreads{})
	spreadJudgingThePeers(network, replicasOfTheTwoQueryWords(), judgements, &recordedSpreads{})

	if amount := amountOfCrossChecksPutTo("third", network.crossCheckedDocumentsAsks); amount != 1 {
		t.Fatalf(
			"the spreads asked the peer to cross-check %d times, want only the ask that judged it",
			amount,
		)
	}
}

func TestAPeerThatListedOnlyTheDocumentsToMatchIsAskedByTheNextQuery(t *testing.T) {
	t.Parallel()

	network := networkOfAReplicaTheFirstRoundLeavesUnasked()
	judgements := judgementsOfTheCrossCheck()
	observer := &recordedSpreads{}

	spreadJudgingThePeers(network, replicasOfTheTwoQueryWords(), judgements, observer)
	spreadJudgingThePeers(network, replicasOfTheTwoQueryWords(), judgements, observer)

	if amount := amountOfCrossChecksPutTo("third", network.crossCheckedDocumentsAsks); amount != 2 {
		t.Fatalf("the spreads asked the peer to cross-check %d times, want once per query", amount)
	}
	if standing := standingReportedFor(t, observer.performed[1], "third"); standing !=
		peerjudgements.Honoring {
		t.Fatalf("the next query reported the peer %q, want honoring", standing)
	}
}

func standingReportedFor(
	t *testing.T,
	performed wordjoined.PerformedWordJoinedSpread,
	address string,
) peerjudgements.Standing {
	t.Helper()

	for _, peerStanding := range performed.PeerStandings {
		if peerStanding.Peer == peerAt(address).Hash {
			return peerStanding.Standing
		}
	}
	t.Fatalf("the spread reported no standing of %q among %v", address, performed.PeerStandings)

	return ""
}

func TestAnEmptyCrossCheckAnswerLeavesThePeerUnjudged(t *testing.T) {
	t.Parallel()

	network := networkOfAReplicaTheFirstRoundLeavesUnasked()
	network.peersHoldingNoDocumentToMatch = map[string]struct{}{"third": {}}
	judgements := judgementsOfTheCrossCheck()
	observer := &recordedSpreads{}

	spreadJudgingThePeers(network, replicasOfTheTwoQueryWords(), judgements, observer)
	spreadJudgingThePeers(network, replicasOfTheTwoQueryWords(), judgements, observer)

	if standing := standingReportedFor(t, observer.performed[1], "third"); standing !=
		peerjudgements.NeverJudged {
		t.Fatalf("the next query reported the peer %q, want never judged", standing)
	}
}

func TestADocumentTheSecondRoundProvesJoinsTheDocumentsOfTheFirst(t *testing.T) {
	t.Parallel()

	network := networkOfAReplicaTheFirstRoundLeavesUnasked()
	observer := &recordedSpreads{}

	spreadTheQuery(firstWord+" "+secondWord, network, replicasOfTheTwoQueryWords(), observer)

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

	spreadTheQuery(firstWord+" "+secondWord, network, peersOfEachQueryWord(map[string][]string{
		firstWord:  {"first"},
		secondWord: {"second"},
	}), observer)

	matchedAndHeldDocumentsRound := observer.performed[0].MatchedAndHeldDocumentsRound
	if matchedAndHeldDocumentsRound.LeadingQueryWordChoice != wordjoined.RarestFullyListedQueryWord ||
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
		"first":  {firstWord: {anchored, unlisted, "https://other.example/"}},
		"fourth": {firstWord: {anchored, unlisted, "https://other.example/"}},
		"second": {secondWord: {
			anchored, unlisted, "https://third.example/",
			"https://fourth.example/", "https://fifth.example/",
		}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"first": 1}
	network.documentsHeldByEachPeer = map[string]int{"second": 5}
	network.replicasPutPerWord = map[string]int{firstWord: 1}
	observer := &recordedSpreads{}

	spreadTheQuery(firstWord+" "+secondWord, network, peersOfEachQueryWord(map[string][]string{
		firstWord:  {"first", "fourth"},
		secondWord: {"second"},
	}), observer)

	performed := observer.performed[0]
	if performed.MatchedAndHeldDocumentsRound.LeadingQueryWordChoice != wordjoined.MoreCommonFullyListedQueryWord ||
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
		"fourth": {
			firstWord: {"https://second.example/", anchored, "https://third.example/"},
		},
		"third": {thirdWord: {
			"https://fourth.example/", anchored, "https://second.example/",
			"https://third.example/", "https://fifth.example/",
		}},
		"fifth": {thirdWord: {anchored}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"first": 1, "second": 1, "third": 1}
	network.replicasPutPerWord = map[string]int{firstWord: 1, thirdWord: 1}
	observer := &recordedSpreads{}

	spreadTheQuery(
		firstWord+" "+secondWord+" "+thirdWord,
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first", "fourth"},
			secondWord: {"second"},
			thirdWord:  {"third", "fifth"},
		}),
		observer,
	)

	matchedAndHeldDocumentsRound := observer.performed[0].MatchedAndHeldDocumentsRound
	if matchedAndHeldDocumentsRound.LeadingQueryWordChoice != wordjoined.RarestPartlyListedQueryWord ||
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
			"the asks named %v, want the document the leading query word listed, once per "+
				"other word",
			got,
		)
	}
}

func TestAReplicaOfTwoPartlyListedQueryWordsIsAskedToCrossCheckOnce(t *testing.T) {
	t.Parallel()

	both := []string{"https://answered.example/", "https://anchored.example/"}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://anchored.example/"}},
		"second": {secondWord: both, thirdWord: both},
		"third":  {secondWord: both, thirdWord: both},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1}
	network.replicasPutPerWord = map[string]int{secondWord: 1, thirdWord: 1}
	observer := &recordedSpreads{}

	spreadTheQuery(
		firstWord+" "+secondWord+" "+thirdWord,
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second", "third"},
			thirdWord:  {"second", "third"},
		}),
		observer,
	)

	if got := addressesAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, []string{"third"},
	) {
		t.Fatalf("the spread asked %v to cross-check, want the unasked replica once", got)
	}
	if amount := observer.performed[0].CrossCheckedDocumentsRound.
		AmountOfCrossCheckCandidatesNoPeerTook; amount != 1 {
		t.Fatalf(
			"the spread reported %d candidates no peer took, want the one of the word left "+
				"with no replica to ask",
			amount,
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
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1}
	network.replicasPutPerWord = map[string]int{secondWord: 1}
	observer := &recordedSpreads{}

	spreadTheQuery(firstWord+" "+secondWord, network, peersOfEachQueryWord(map[string][]string{
		firstWord:  {"first", "fourth", "fifth"},
		secondWord: {"second", "third"},
	}), observer)

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
	if crossCheckedDocumentsRound.AmountOfCrossCheckCandidatesNoPeerTook != 0 ||
		crossCheckedDocumentsRound.AmountOfJoinedDocuments != 3 {
		t.Fatalf(
			"the spread reported %+v, want every document listed for the leading query word joined and no candidate that no peer took",
			crossCheckedDocumentsRound,
		)
	}
}

func TestEveryReplicaTheFirstRoundLeftUnaskedIsAskedTheSameDocuments(t *testing.T) {
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
		"fourth": {secondWord: documentsListedForTheLeadingQueryWord},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 1}
	network.replicasPutPerWord = map[string]int{secondWord: 1}

	spreadTheQuery(firstWord+" "+secondWord, network, peersOfEachQueryWord(map[string][]string{
		firstWord:  {"first"},
		secondWord: {"second", "third", "fourth"},
	}), &recordedSpreads{})

	if got := addressesAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, []string{"third", "fourth"},
	) {
		t.Fatalf("the spread asked %v to cross-check, want each replica left unasked", got)
	}
	wanted := documentsInTheirHashOrder(documentHashesOf(documentsListedForTheLeadingQueryWord[1:]))
	for _, ask := range network.crossCheckedDocumentsAsks {
		if got := documentsInTheirHashOrder(ask.DocumentsToMatch); !slices.Equal(got, wanted) {
			t.Fatalf(
				"the ask to peer %q named %v, want every candidate %v",
				ask.Peer.Address,
				got,
				wanted,
			)
		}
	}
}

func TestTheCandidatesOverTheCeilingAreCountedAsNoPeerTook(
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
		"fourth": {secondWord: documentsListedForTheLeadingQueryWord},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 0}
	network.replicasPutPerWord = map[string]int{secondWord: 1}
	observer := &recordedSpreads{}

	spreadUnder(
		documentsOneCrossCheckedDocumentsAskNames,
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second", "third", "fourth"},
		}),
		observer,
	)

	crossCheckedDocumentsRound := observer.performed[0].CrossCheckedDocumentsRound
	if crossCheckedDocumentsRound.AmountOfCrossCheckCandidatesNoPeerTook != 3 ||
		crossCheckedDocumentsRound.AmountOfDocumentsSentForCrossChecking != 1 {
		t.Fatalf(
			"the spread reported %+v, want the one document the ceiling allows sent and the "+
				"three over the ceiling counted as no peer took",
			crossCheckedDocumentsRound,
		)
	}
	if len(network.crossCheckedDocumentsAsks) != 2 {
		t.Fatalf(
			"the spread put %v, want a cross-checked documents ask to each replica left unasked",
			network.crossCheckedDocumentsAsks,
		)
	}
	firstAsk, secondAsk := network.crossCheckedDocumentsAsks[0], network.crossCheckedDocumentsAsks[1]
	if len(firstAsk.DocumentsToMatch) != documentsOneCrossCheckedDocumentsAskNames ||
		!slices.Equal(firstAsk.DocumentsToMatch, secondAsk.DocumentsToMatch) {
		t.Fatalf(
			"the asks named %v and %v, want the same one document the ceiling allows",
			firstAsk.DocumentsToMatch,
			secondAsk.DocumentsToMatch,
		)
	}
}

func TestTheCeilingKeepsTheCandidatesListedByTheMostPeers(t *testing.T) {
	t.Parallel()

	listedByBoth := "https://listed-by-both.example/"
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://listed-by-one.example/", listedByBoth}},
		"fourth": {firstWord: {listedByBoth}},
		"second": {secondWord: {"https://listed-by-one.example/", listedByBoth}},
		"third":  {secondWord: {"https://listed-by-one.example/", listedByBoth}},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 0}
	network.replicasPutPerWord = map[string]int{secondWord: 1}

	spreadUnder(1, network, peersOfEachQueryWord(map[string][]string{
		firstWord:  {"first", "fourth"},
		secondWord: {"second", "third"},
	}), &recordedSpreads{})

	wanted := documentHashesOf([]string{listedByBoth})
	if got := documentsAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, wanted,
	) {
		t.Fatalf("the asks named %v, want only the candidate both peers listed %v", got, wanted)
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
		"fourth": {secondWord: documentsListedForTheLeadingQueryWord},
		"third":  {thirdWord: documentsListedForTheLeadingQueryWord},
		"fifth":  {thirdWord: documentsListedForTheLeadingQueryWord},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second": 0, "third": 0}
	network.replicasPutPerWord = map[string]int{secondWord: 1, thirdWord: 1}
	observer := &recordedSpreads{}

	spreadTheQuery(
		firstWord+" "+secondWord+" "+thirdWord,
		network,
		peersOfEachQueryWord(map[string][]string{
			firstWord:  {"first"},
			secondWord: {"second", "fourth"},
			thirdWord:  {"third", "fifth"},
		}),
		observer,
	)

	crossCheckedDocumentsRound := observer.performed[0].CrossCheckedDocumentsRound
	if crossCheckedDocumentsRound.AmountOfDocumentsSentForCrossChecking != 4 ||
		crossCheckedDocumentsRound.AmountOfCrossCheckCandidatesNoPeerTook != 0 {
		t.Fatalf(
			"the spread reported %+v, want both documents sent once for each of the two other "+
				"query words",
			crossCheckedDocumentsRound,
		)
	}
}

func TestAPeerThatHoldsNoneOfTheDocumentsToMatchLeavesTheJoinOfTheFirstRoundWhole(t *testing.T) {
	t.Parallel()

	network := networkOfAReplicaTheFirstRoundLeavesUnasked()
	network.peersHoldingNoDocumentToMatch = map[string]struct{}{"third": {}}
	observer := &recordedSpreads{}

	spreadTheQuery(firstWord+" "+secondWord, network, replicasOfTheTwoQueryWords(), observer)

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

	answeredQueryFrom(network, observer)

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

func TestAPeerThatStayedSilentInTheFirstRoundIsNotAskedAgain(t *testing.T) {
	t.Parallel()

	shared := []string{"https://shared.example/"}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: shared},
		"second": {secondWord: shared},
	})
	network.silentPeers["second"] = struct{}{}

	answeredQueryWith(peersOfEachQueryWord(map[string][]string{
		firstWord:  {"first"},
		secondWord: {"second"},
	}), network, &recordedSpreads{})

	if len(network.crossCheckedDocumentsAsks) != 0 || len(network.urlMetadataAsks) != 0 {
		t.Fatalf(
			"the spread put %v and %v, want no second call to the peer the first round asked",
			network.crossCheckedDocumentsAsks,
			network.urlMetadataAsks,
		)
	}
}

func TestTheFirstRoundKeepsOnlyAThirdOfTheTimeTheQueryHasLeft(t *testing.T) {
	t.Parallel()

	const queryBudget = 3 * time.Second

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://shared.example/"}},
		"second": {secondWord: {"https://shared.example/"}},
	})

	spreadWithin(queryBudget, network)

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

	answeredQueryFrom(network, &recordedSpreads{})

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

	answeredQueryFrom(network, &recordedSpreads{})

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

	answeredQueryFrom(network, &recordedSpreads{})

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

	foundDocuments := answeredQueryFrom(network, &recordedSpreads{}).FoundDocuments

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

	answeredQueryWith(responsiblePeers{peerAddressesPerWord: map[string][]string{
		firstWord:  {"first"},
		secondWord: {"second"},
	}}, network, observer)

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

	answeredQueryFrom(network, observer)

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

	answeredQueryFrom(network, observer)

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

	answeredQueryFrom(network, observer)

	if len(network.urlMetadataAsks) != 0 {
		t.Fatalf(
			"asked %d peers about documents, want none once a word went unanswered",
			len(network.urlMetadataAsks),
		)
	}
}

func TestNoPeerIsAskedMetadataForMoreDocumentsThanTheCeiling(t *testing.T) {
	t.Parallel()

	heldByFirst := []string{
		"https://first.example/",
		"https://second.example/",
		"https://third.example/",
	}
	heldBySecond := []string{
		"https://fourth.example/",
		"https://fifth.example/",
		"https://sixth.example/",
	}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: heldByFirst, secondWord: heldByFirst},
		"second": {firstWord: heldBySecond, secondWord: heldBySecond},
	})

	answeredQueryUnder(
		documentsOneURLMetadataAskNames,
		network,
		responsiblePeers{},
		&recordedSpreads{},
	)

	if len(network.urlMetadataAsks) != 2 {
		t.Fatalf("%d peers were asked for metadata, want both", len(network.urlMetadataAsks))
	}
	for _, ask := range network.urlMetadataAsks {
		if len(ask.Documents) != 1 {
			t.Fatalf(
				"peer %q was asked metadata for %d documents, want the one the ceiling allows",
				ask.Peer.Address,
				len(ask.Documents),
			)
		}
	}
}

func TestTheDocumentsTheMostPeersHoldAreTheOnesEachPeerIsAskedMetadataFor(t *testing.T) {
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

	answeredQueryUnder(
		documentsOneURLMetadataAskNames,
		network,
		responsiblePeers{},
		&recordedSpreads{},
	)

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

	answeredQueryUnder(documentsOneURLMetadataAskNames, network, responsiblePeers{}, observer)

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
		newSpreadOverChosenPeers(
			responsiblePeers{},
			wordjoined.New(
				network,
				network,
				judgementsOfTheCrossCheck(),
				urlMetadataAskDocumentsCeiling,
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

	answeredQueryFrom(network, &recordedSpreads{})

	wanted := documentHashesOf([]string{unanswered})
	if got := distinctDocumentsAskedMetadataFor(network.urlMetadataAsks); !slices.Equal(
		got, wanted,
	) {
		t.Fatalf("asked about %v, want only the joined document no peer answered", got)
	}
}

func TestAJoinedDocumentACrossCheckAnswerMatchedIsFoundAndNotAskedMetadataFor(t *testing.T) {
	t.Parallel()

	network := networkOfAReplicaTheFirstRoundLeavesUnasked()
	network.countsAWordWithEachItem = true

	foundDocuments := answeredQueryUnder(
		urlMetadataAskDocumentsCeiling, network, replicasOfTheTwoQueryWords(), &recordedSpreads{},
	).FoundDocuments

	crossChecked := documentHashOf(t, "https://anchored.example/")
	if slices.Contains(distinctDocumentsAskedMetadataFor(network.urlMetadataAsks), crossChecked) {
		t.Fatalf("asked metadata for %v, want none for the document the cross-check matched",
			network.urlMetadataAsks)
	}
	place := slices.IndexFunc(foundDocuments, func(foundDocument queryanswers.FoundDocument) bool {
		return foundDocument.Hash == crossChecked
	})
	if place < 0 {
		t.Fatalf("the spread found %+v, want the document the cross-check matched", foundDocuments)
	}
	if hits := foundDocuments[place].Facts.HitsPerQueryWord[yacymodel.WordHash(secondWord)]; hits != 3 {
		t.Fatalf("the found document holds the hits %v, want the hits the cross-check counted",
			foundDocuments[place].Facts.HitsPerQueryWord)
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

	foundDocuments := answeredQueryFrom(network, &recordedSpreads{}).FoundDocuments

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

	answers := answeredQueryFrom(network, &recordedSpreads{})

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

	answers := answeredQueryFrom(network, &recordedSpreads{})
	foundDocuments := answers.FoundDocuments

	if len(foundDocuments) != 1 {
		t.Fatalf("the spread found %v, want the one document the peer answered", foundDocuments)
	}
	hitsPerQueryWord := foundDocuments[0].Facts.HitsPerQueryWord
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

	answers := answeredQueryFrom(network, &recordedSpreads{})
	foundDocuments := answers.FoundDocuments

	if len(foundDocuments) != 1 {
		t.Fatalf("the spread found %v, want the joined document once", foundDocuments)
	}
	hitsPerQueryWord := foundDocuments[0].Facts.HitsPerQueryWord
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

	foundDocuments := answeredQueryFrom(network, &recordedSpreads{}).FoundDocuments

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

	answers := answeredQueryFrom(network, &recordedSpreads{})

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

	answers := answeredQueryFrom(network, &recordedSpreads{})

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
		newSpreadOverChosenPeers(
			choice,
			wordjoined.New(
				network,
				network,
				judgementsOfTheCrossCheck(),
				urlMetadataAskDocumentsCeiling,
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

	answeredQueryWith(responsiblePeers{peerAddressesPerWord: map[string][]string{
		firstWord:  {"first"},
		secondWord: {"first"},
	}}, network, observer)

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
		newSpreadOverChosenPeers(
			choice,
			wordjoined.New(
				network,
				network,
				judgementsOfTheCrossCheck(),
				urlMetadataAskDocumentsCeiling,
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
	asks []peerasks.SearchDocumentsAsk,
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

type spreadOverChosenPeers struct {
	responsiblePeers responsiblePeers
	wordJoinedSpread wordjoined.Spread
}

func newSpreadOverChosenPeers(
	responsiblePeers responsiblePeers,
	wordJoinedSpread wordjoined.Spread,
) spreadOverChosenPeers {
	return spreadOverChosenPeers{
		responsiblePeers: responsiblePeers,
		wordJoinedSpread: wordJoinedSpread,
	}
}

func (spread spreadOverChosenPeers) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) queryanswers.AnsweredQuery {
	return spread.wordJoinedSpread.SpreadOverPeers(
		ctx,
		query,
		spread.responsiblePeers.ChosenPeersPerQueryWordFor(
			ctx, query.HashesOfWordsAndCompoundWordsUpTo(compoundWordsCeiling), askablePeers,
		),
	)
}

func TestAFoundDocumentCarriesTheAmountOfLinksThePostingReported(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	})
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"first": {firstWord: {answered}},
	}
	network.countsAWordWithEachItem = true

	answers := answeredQueryFrom(network, &recordedSpreads{})
	foundDocuments := answers.FoundDocuments

	if len(foundDocuments) != 1 {
		t.Fatalf("the spread found %v, want the one document the peer answered", foundDocuments)
	}
	amountOfLinks, reported := foundDocuments[0].Facts.AmountOfLinks.Get()
	if !reported || amountOfLinks != 19 {
		t.Fatalf(
			"the found document holds %d links reported %t, want the 19 links the posting reported",
			amountOfLinks,
			reported,
		)
	}
}

func TestAJoinedDocumentFoundThroughItsMetadataAloneHoldsNoAmountOfLinks(t *testing.T) {
	t.Parallel()

	joined := "https://joined.example/"
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {joined}, secondWord: {joined}},
		"second": {firstWord: {joined}, secondWord: {joined}},
	})

	answers := answeredQueryFrom(network, &recordedSpreads{})
	foundDocuments := answers.FoundDocuments

	if len(foundDocuments) != 1 {
		t.Fatalf("the spread found %v, want the joined document once", foundDocuments)
	}
	if foundDocuments[0].Facts.AmountOfLinks.Present() {
		t.Fatal("the joined document holds an amount of links, want none where no posting " +
			"reported one")
	}
}

func TestADocumentListedForTheCompoundWordOfTwoWordsIsJoinedForBoth(t *testing.T) {
	t.Parallel()

	compounded := "https://compounded.example/"
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://half.example/"}},
		"second": {firstWord + secondWord: {compounded}},
	})

	answers := answeredQueryFrom(network, &recordedSpreads{})

	if len(answers.FoundDocuments) != 1 ||
		answers.FoundDocuments[0].Hash != documentHashesOf([]string{compounded})[0] {
		t.Fatalf(
			"the spread found %+v, want the one document listed for the compound word",
			answers.FoundDocuments,
		)
	}
}

func addressesInPartition(
	t *testing.T,
	partition uint,
	amount int,
) []string {
	t.Helper()

	addresses := make([]string, 0, amount)
	for place := 0; len(addresses) < amount; place++ {
		address := fmt.Sprintf("https://document-%d.example/", place)
		if yacymodel.DHTRingPartitions(twoPartitionsOfTheRing).PartitionOf(
			documentHashOf(t, address),
		) != partition {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func replicasOfTwoQueryWordsInTwoPartitions() responsiblePeers {
	return responsiblePeers{
		peerAddressesPerWord: map[string][]string{
			firstWord: {"first-in-0", "first-in-1"},
			secondWord: {
				"second-in-0", "second-in-1", "unasked-second-in-0", "unasked-second-in-1",
			},
		},
		partitionOfEachPeer: map[string]uint{
			"first-in-0":          0,
			"first-in-1":          1,
			"second-in-0":         0,
			"second-in-1":         1,
			"unasked-second-in-0": 0,
			"unasked-second-in-1": 1,
		},
	}
}

func spreadOverTwoPartitions(
	network *peerNetwork,
	choice responsiblePeers,
	judgements wordjoined.PeerJudgements,
	observer wordjoined.WordJoinedSpreadObserver,
) queryanswers.AnsweredQuery {
	return spreadOverPeers(
		newSpreadOverChosenPeers(
			choice,
			wordjoined.New(
				network,
				network,
				judgements,
				urlMetadataAskDocumentsCeiling,
				crossCheckedDocumentsCeiling,
				peerItemsCeiling,
				twoPartitionsOfTheRing,
				peersHoldingOneWord,
				observer,
			),
		),
		nil,
	)
}

func networkWhereTheSecondWordHoldsTheDocumentsOfPartitionOne(
	t *testing.T,
) (*peerNetwork, []string) {
	t.Helper()

	documentsInPartitionOne := addressesInPartition(t, 1, 3)
	network := networkOf(map[string]map[string][]string{
		"first-in-0":          {firstWord: {}},
		"first-in-1":          {firstWord: documentsInPartitionOne[:2]},
		"second-in-0":         {secondWord: addressesInPartition(t, 0, 1)},
		"second-in-1":         {secondWord: documentsInPartitionOne},
		"unasked-second-in-0": {secondWord: addressesInPartition(t, 0, 1)},
		"unasked-second-in-1": {secondWord: documentsInPartitionOne},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second-in-0": 0, "second-in-1": 0}
	network.replicasPutPerWord = map[string]int{secondWord: 1}

	return network, documentsInPartitionOne[:2]
}

func TestACandidateOfOnePartitionIsAskedOnlyOfTheReplicasInThatPartition(t *testing.T) {
	t.Parallel()

	network, candidates := networkWhereTheSecondWordHoldsTheDocumentsOfPartitionOne(t)

	spreadOverTwoPartitions(
		network,
		replicasOfTwoQueryWordsInTwoPartitions(),
		judgementsOfTheCrossCheck(),
		&recordedSpreads{},
	)

	if got := addressesAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, []string{"unasked-second-in-1"},
	) {
		t.Fatalf(
			"the spread asked %v to cross-check, want the unasked replica in partition 1",
			got,
		)
	}
	wanted := documentsInTheirHashOrder(documentHashesOf(candidates))
	got := documentsInTheirHashOrder(documentsAskedToCrossCheck(network.crossCheckedDocumentsAsks))
	if !slices.Equal(got, wanted) || network.crossCheckedDocumentsAsks[0].Partition != 1 {
		t.Fatalf("the ask named %v, want every candidate of partition 1 %v", got, wanted)
	}
}

func TestADocumentOnlyAReplicaInItsPartitionHoldsIsFound(t *testing.T) {
	t.Parallel()

	network, candidates := networkWhereTheSecondWordHoldsTheDocumentsOfPartitionOne(t)

	answeredQuery := spreadOverTwoPartitions(
		network,
		replicasOfTwoQueryWordsInTwoPartitions(),
		judgementsOfTheCrossCheck(),
		&recordedSpreads{},
	)

	foundDocuments := make([]yacymodel.URLHash, 0, len(answeredQuery.FoundDocuments))
	for _, foundDocument := range answeredQuery.FoundDocuments {
		foundDocuments = append(foundDocuments, foundDocument.Hash)
	}
	wanted := documentsInTheirHashOrder(documentHashesOf(candidates))
	if got := documentsInTheirHashOrder(foundDocuments); !slices.Equal(got, wanted) {
		t.Fatalf("the spread found %v, want every document the replica in partition 1 holds %v",
			got, wanted)
	}
}

func networkWhereTheSecondWordIsFullyListedInPartitionZero(
	t *testing.T,
) (*peerNetwork, []string) {
	t.Helper()

	documentsInPartitionZero := addressesInPartition(t, 0, 2)
	documentsInPartitionOne := addressesInPartition(t, 1, 2)
	secondWordInPartitionOne := []string{documentsInPartitionOne[1], documentsInPartitionOne[0]}
	network := networkOf(map[string]map[string][]string{
		"first-in-0":          {firstWord: documentsInPartitionZero[:1]},
		"first-in-1":          {firstWord: documentsInPartitionOne[:1]},
		"second-in-0":         {secondWord: documentsInPartitionZero[1:]},
		"second-in-1":         {secondWord: secondWordInPartitionOne},
		"unasked-second-in-1": {secondWord: secondWordInPartitionOne},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second-in-1": 1}
	network.documentsHeldByEachPeer = map[string]int{"second-in-0": 1}
	network.replicasPutPerWord = map[string]int{secondWord: 1}

	return network, documentsInPartitionOne
}

func TestNoCandidateOfAPartitionWhereTheWordIsFullyListedIsAsked(t *testing.T) {
	t.Parallel()

	network, documentsInPartitionOne := networkWhereTheSecondWordIsFullyListedInPartitionZero(t)

	spreadOverTwoPartitions(
		network,
		replicasOfTwoQueryWordsInTwoPartitions(),
		judgementsOfTheCrossCheck(),
		&recordedSpreads{},
	)

	wanted := documentHashesOf(documentsInPartitionOne[:1])
	if got := documentsAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, wanted,
	) {
		t.Fatalf(
			"the asks named %v, want only the candidate of the partition the word is partly "+
				"listed in %v",
			got,
			wanted,
		)
	}
}

func TestTheCandidatesOfAPartitionWhereTheWordIsFullyListedAreReportedAsRuledOut(t *testing.T) {
	t.Parallel()

	network, _ := networkWhereTheSecondWordIsFullyListedInPartitionZero(t)
	observer := &recordedSpreads{}

	spreadOverTwoPartitions(
		network,
		replicasOfTwoQueryWordsInTwoPartitions(),
		judgementsOfTheCrossCheck(),
		observer,
	)

	crossCheckedDocumentsRound := observer.performed[0].CrossCheckedDocumentsRound
	if crossCheckedDocumentsRound.AmountOfCrossCheckCandidatesRuledOutByAFullListing != 1 ||
		crossCheckedDocumentsRound.AmountOfDocumentsSentForCrossChecking != 1 ||
		crossCheckedDocumentsRound.AmountOfCrossCheckCandidatesNoPeerTook != 0 {
		t.Fatalf(
			"the spread reported %+v, want the candidate of partition 0 ruled out and the one of "+
				"partition 1 sent",
			crossCheckedDocumentsRound,
		)
	}
}

func TestTheCandidatesOfAPartitionWithNoReplicaLeftToAskAreCountedAsNoPeerTook(
	t *testing.T,
) {
	t.Parallel()

	documentsInPartitionZero := addressesInPartition(t, 0, 1)
	documentsInPartitionOne := addressesInPartition(t, 1, 2)
	network := networkOf(map[string]map[string][]string{
		"first-in-0":          {firstWord: documentsInPartitionZero},
		"first-in-1":          {firstWord: documentsInPartitionOne[:1]},
		"second-in-0":         {secondWord: documentsInPartitionZero},
		"second-in-1":         {secondWord: documentsInPartitionOne},
		"unasked-second-in-0": {secondWord: documentsInPartitionZero},
		"unasked-second-in-1": {secondWord: documentsInPartitionOne},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"second-in-0": 0, "second-in-1": 0}
	network.replicasPutPerWord = map[string]int{secondWord: 1}
	observer := &recordedSpreads{}

	spreadOverTwoPartitions(
		network,
		replicasOfTwoQueryWordsInTwoPartitions(),
		judgementsWhere(
			peerJudged("second-in-1", peerjudgements.Ignored),
			peerJudged("unasked-second-in-1", peerjudgements.Ignored),
		),
		observer,
	)

	if got := addressesAskedToCrossCheck(network.crossCheckedDocumentsAsks); !slices.Equal(
		got, []string{"unasked-second-in-0"},
	) {
		t.Fatalf(
			"the spread asked %v to cross-check, want only the replica in partition 0 left to ask",
			got,
		)
	}
	crossCheckedDocumentsRound := observer.performed[0].CrossCheckedDocumentsRound
	if crossCheckedDocumentsRound.AmountOfCrossCheckCandidatesNoPeerTook != 1 {
		t.Fatalf(
			"the spread reported %+v, want the one candidate of partition 1 that no peer took",
			crossCheckedDocumentsRound,
		)
	}
}

type recordedJudgements struct {
	wordjoined.PeerJudgements

	amountOfLookups int
	peersLookedUp   []peerjudgements.PeerAtVersion
}

func (r *recordedJudgements) StandingsOf(
	ctx context.Context,
	peers []peerjudgements.PeerAtVersion,
) peerjudgements.PeerStandings {
	r.amountOfLookups++
	r.peersLookedUp = append(r.peersLookedUp, peers...)

	return r.PeerJudgements.StandingsOf(ctx, peers)
}

func TestAPartitionWithABetterLeadingQueryWordIsReported(
	t *testing.T,
) {
	t.Parallel()

	network := networkOf(map[string]map[string][]string{
		"first-in-0":  {firstWord: addressesInPartition(t, 0, 1)},
		"first-in-1":  {firstWord: addressesInPartition(t, 1, 2)},
		"second-in-0": {secondWord: addressesInPartition(t, 0, 5)},
		"second-in-1": {secondWord: addressesInPartition(t, 1, 1)},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"first-in-1": 0, "second-in-0": 0}
	network.documentsHeldByEachPeer = map[string]int{"first-in-0": 1, "second-in-1": 1}
	network.replicasPutPerWord = map[string]int{secondWord: 1}
	observer := &recordedSpreads{}

	spreadOverTwoPartitions(
		network,
		replicasOfTwoQueryWordsInTwoPartitions(),
		judgementsOfTheCrossCheck(),
		observer,
	)

	matchedAndHeldDocumentsRound := observer.performed[0].MatchedAndHeldDocumentsRound
	if matchedAndHeldDocumentsRound.LeadingQueryWordChoice != wordjoined.RarestPartlyListedQueryWord ||
		matchedAndHeldDocumentsRound.AmountOfPartitionsWithABetterLeadingQueryWord != 1 {
		t.Fatalf(
			"the spread reported %+v, want the rarest word leading and partition 1, where the "+
				"other word is fully listed, as the one with a better leading word",
			matchedAndHeldDocumentsRound,
		)
	}
}
