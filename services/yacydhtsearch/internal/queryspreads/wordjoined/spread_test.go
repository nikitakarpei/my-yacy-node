package wordjoined_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord                       = "berlin"
	secondWord                      = "weather"
	thirdWord                       = "rain"
	urlMetadataAskDocumentsCeiling  = 10
	documentsOneURLMetadataAskNames = 1
	documentsToMatchCeiling         = 10
	peersHoldingOneWord             = 24
	onePartitionOfTheRing           = 1
	compoundWordsCeiling            = 4
	twoPartitionsOfTheRing          = 2
)

type peerNetwork struct {
	mutex                                 sync.Mutex
	documentsPerWordPerPeer               map[string]map[string][]string
	replicaAsks                           []replicaAsk
	settledWordPartitionsReadAtEachAsk    []int
	urlMetadataAsks                       []peerasks.URLMetadataAsk
	silentPeers                           map[string]struct{}
	peersListingDocumentsTheAskDidNotName map[string]struct{}
	timeLeftAtEachCallInTheirOrder        []time.Duration
	metadataDelayOfEachPeer               map[string]time.Duration
	peersFailingTheURLMetadataCall        map[string]struct{}
	urlMetadataLookupContextOfEachPeer    map[string]context.Context
}

func networkOf(documentsPerWordPerPeer map[string]map[string][]string) *peerNetwork {
	return &peerNetwork{
		documentsPerWordPerPeer:               documentsPerWordPerPeer,
		silentPeers:                           map[string]struct{}{},
		peersListingDocumentsTheAskDidNotName: map[string]struct{}{},
		metadataDelayOfEachPeer:               map[string]time.Duration{},
		peersFailingTheURLMetadataCall:        map[string]struct{}{},
		urlMetadataLookupContextOfEachPeer:    map[string]context.Context{},
	}
}

type replicasOfTheNetwork struct {
	network *peerNetwork
}

func replicasOf(network *peerNetwork) replicasOfTheNetwork {
	return replicasOfTheNetwork{network: network}
}

func (replicas replicasOfTheNetwork) Start(ctx context.Context) wordpartitionasks.Run {
	asks := make(chan []wordpartitionasks.Ask)
	settledAsks := make(chan wordpartitionasks.SettledAsk)
	go replicas.answerEachWordPartition(ctx, asks, settledAsks)

	return wordpartitionasks.Run{Asks: asks, SettledAsks: settledAsks}
}

func (replicas replicasOfTheNetwork) answerEachWordPartition(
	ctx context.Context,
	asks <-chan []wordpartitionasks.Ask,
	settledAsks chan<- wordpartitionasks.SettledAsk,
) {
	defer close(settledAsks)
	run := &runOfTheNetwork{wordPartitionsInTheRun: map[string]struct{}{}}
	for asks != nil || len(run.settledAsksUnread) > 0 {
		var reader chan<- wordpartitionasks.SettledAsk
		var nextSettledAsk wordpartitionasks.SettledAsk
		if len(run.settledAsksUnread) > 0 {
			reader = settledAsks
			nextSettledAsk = run.settledAsksUnread[0]
		}
		select {
		case addedAsks, open := <-asks:
			if !open {
				asks = nil

				continue
			}
			run.answer(ctx, replicas.network, addedAsks)
		case reader <- nextSettledAsk:
			run.settledAsksUnread = run.settledAsksUnread[1:]
			run.settledAsksRead++
		}
	}
}

type runOfTheNetwork struct {
	wordPartitionsInTheRun map[string]struct{}
	settledAsksUnread      []wordpartitionasks.SettledAsk
	settledAsksRead        int
}

func (run *runOfTheNetwork) answer(
	ctx context.Context,
	network *peerNetwork,
	asks []wordpartitionasks.Ask,
) {
	for _, ask := range asks {
		wordPartition := fmt.Sprintf("%s in %d", ask.Word, ask.Partition)
		if _, inTheRun := run.wordPartitionsInTheRun[wordPartition]; inTheRun {
			continue
		}
		run.wordPartitionsInTheRun[wordPartition] = struct{}{}
		settledAsk := wordpartitionasks.SettledAsk{Ask: ask}
		for _, replica := range ask.ReplicasInOrder {
			answer, answered := network.answerOfTheReplica(
				ctx, replicaAsk{Ask: ask, Peer: replica}, run.settledAsksRead,
			)
			if !answered {
				continue
			}
			settledAsk.Answers = append(settledAsk.Answers, answer)
		}
		run.settledAsksUnread = append(run.settledAsksUnread, settledAsk)
	}
}

type replicaAsk struct {
	wordpartitionasks.Ask
	Peer peerdirectory.AskablePeer
}

func (network *peerNetwork) answerOfTheReplica(
	ctx context.Context,
	ask replicaAsk,
	settledAsksRead int,
) (wordpartitionasks.ReplicaAnswer, bool) {
	network.mutex.Lock()
	defer network.mutex.Unlock()

	network.replicaAsks = append(network.replicaAsks, ask)
	network.settledWordPartitionsReadAtEachAsk = append(
		network.settledWordPartitionsReadAtEachAsk, settledAsksRead,
	)
	network.recordTimeLeftIn(ctx)
	if _, silent := network.silentPeers[ask.Peer.Address]; silent {
		return wordpartitionasks.ReplicaAnswer{}, false
	}

	return wordpartitionasks.ReplicaAnswer{
		Replica:         ask.Peer,
		ListedDocuments: listedDocumentsOf(network.abstractFor(ask)),
	}, true
}

func listedDocumentsOf(abstract []yacymodel.URLHash) []wordpartitionasks.ListedDocument {
	listedDocuments := make([]wordpartitionasks.ListedDocument, 0, len(abstract))
	for _, document := range abstract {
		listedDocuments = append(listedDocuments, wordpartitionasks.ListedDocument{Hash: document})
	}

	return listedDocuments
}

func (network *peerNetwork) recordTimeLeftIn(ctx context.Context) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		network.timeLeftAtEachCallInTheirOrder = append(network.timeLeftAtEachCallInTheirOrder, 0)

		return
	}
	network.timeLeftAtEachCallInTheirOrder = append(
		network.timeLeftAtEachCallInTheirOrder, time.Until(deadline),
	)
}

func (network *peerNetwork) abstractFor(ask replicaAsk) []yacymodel.URLHash {
	documents := documentsPerWordOf(network.documentsPerWordPerPeer, ask.Peer.Address, ask.Word)
	_, listsMore := network.peersListingDocumentsTheAskDidNotName[ask.Peer.Address]
	if len(ask.DocumentsToMatch) == 0 || listsMore {
		return documents
	}

	namedDocuments := make([]yacymodel.URLHash, 0, len(ask.DocumentsToMatch))
	for _, document := range documents {
		if !slices.Contains(ask.DocumentsToMatch, document) {
			continue
		}
		namedDocuments = append(namedDocuments, document)
	}

	return namedDocuments
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

func (network *peerNetwork) AskForURLMetadata(
	ctx context.Context,
	asks []peerasks.URLMetadataAsk,
) <-chan peerasks.URLMetadataAskOutcome {
	network.mutex.Lock()
	defer network.mutex.Unlock()

	network.urlMetadataAsks = append(network.urlMetadataAsks, asks...)
	network.recordTimeLeftIn(ctx)

	outcomesAsTheySettle := make(chan peerasks.URLMetadataAskOutcome, len(asks))
	var askSettlings sync.WaitGroup
	for _, ask := range asks {
		network.urlMetadataLookupContextOfEachPeer[ask.Peer.Address] = ctx
		delay := network.metadataDelayOfEachPeer[ask.Peer.Address]
		outcome := network.outcomeOfTheURLMetadataAsk(ask)
		askSettlings.Go(func() {
			select {
			case <-ctx.Done():
			case <-time.After(delay):
				outcomesAsTheySettle <- outcome
			}
		})
	}
	go func() {
		askSettlings.Wait()
		close(outcomesAsTheySettle)
	}()

	return outcomesAsTheySettle
}

func (network *peerNetwork) outcomeOfTheURLMetadataAsk(
	ask peerasks.URLMetadataAsk,
) peerasks.URLMetadataAskOutcome {
	outcome := peerasks.URLMetadataAskOutcome{Ask: ask, Put: true}
	if _, fails := network.peersFailingTheURLMetadataCall[ask.Peer.Address]; fails {
		return outcome
	}
	outcome.Answer = yacymodel.Some(peerasks.AnsweredURLMetadataAsk{
		Ask:                    ask,
		MetadataOfEachDocument: metadataOfEachDocument(ask.Documents),
	})

	return outcome
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

func (choice responsiblePeers) ChosenPeersPerQueryWordFor(
	queryWords []yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
) peerchoice.ChosenPeersPerQueryWord {
	peersPerQueryWord := make(peerchoice.ChosenPeersPerQueryWord, 0, len(queryWords))
	for _, queryWord := range queryWords {
		peersPerQueryWord = append(peersPerQueryWord, peerchoice.ChosenPeersOfQueryWord{
			QueryWord:   queryWord,
			ChosenPeers: choice.chosenPeersOf(choice.peersForWord(queryWord, askablePeers)),
		})
	}

	return peersPerQueryWord
}

func (choice responsiblePeers) chosenPeersOf(
	askablePeers []peerdirectory.AskablePeer,
) []peerchoice.ChosenPeer {
	chosenPeers := make([]peerchoice.ChosenPeer, 0, len(askablePeers))
	for _, peer := range askablePeers {
		chosenPeers = append(chosenPeers, peerchoice.ChosenPeer{
			Peer:      peer,
			Partition: choice.partitionOfEachPeer[peer.Address],
		})
	}

	return chosenPeers
}

func (choice responsiblePeers) peersForWord(
	word yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	if choice.peerAddressesPerWord == nil {
		return askablePeers
	}
	for spelledWord, addresses := range choice.peerAddressesPerWord {
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

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{
		Hash:    yacymodel.WordHash(address),
		Address: address,
	}
}

type recordedSpreads struct {
	performed []wordjoined.PerformedWordJoinedSpread
}

func (recorded *recordedSpreads) WordJoinedSpreadPerformed(
	_ context.Context,
	spread wordjoined.PerformedWordJoinedSpread,
) {
	recorded.performed = append(recorded.performed, spread)
}

type spreadSettings struct {
	query                           string
	choice                          responsiblePeers
	askablePeers                    []string
	partitions                      yacymodel.DHTRingPartitions
	sampledPartition                uint
	documentsToMatchCeiling         int
	urlMetadataAskDocumentsCeiling  int
	urlMetadataAskCeilingOfEachPeer map[string]int
	peersHoldingOneWord             int
	queryBudget                     time.Duration
	queryWordDocumentAmounts        *rememberedQueryWordDocumentAmounts
	urlMetadataLookupCutoff         wordjoined.URLMetadataLookupCutoff
}

type rememberedQueryWordDocumentAmounts struct {
	amountOfEachWord map[yacymodel.Hash]int
}

func queryWordDocumentAmountsOf(
	amountOfEachWord map[string]int,
) *rememberedQueryWordDocumentAmounts {
	remembered := &rememberedQueryWordDocumentAmounts{amountOfEachWord: map[yacymodel.Hash]int{}}
	for spelledWord, amount := range amountOfEachWord {
		remembered.amountOfEachWord[yacymodel.WordHash(spelledWord)] = amount
	}

	return remembered
}

func (remembered *rememberedQueryWordDocumentAmounts) DocumentAmountsOf(
	_ context.Context,
	words []yacymodel.Hash,
) map[yacymodel.Hash]int {
	amounts := map[yacymodel.Hash]int{}
	for _, word := range words {
		if amount, known := remembered.amountOfEachWord[word]; known {
			amounts[word] = amount
		}
	}

	return amounts
}

func (remembered *rememberedQueryWordDocumentAmounts) Remember(
	_ context.Context,
	documentAmounts map[yacymodel.Hash]int,
) {
	maps.Copy(remembered.amountOfEachWord, documentAmounts)
}

type urlMetadataAskCeilingsOfThePeers struct {
	mostDocuments                   int
	urlMetadataAskCeilingOfEachPeer map[string]int
}

func (ceilings urlMetadataAskCeilingsOfThePeers) CeilingOf(_ context.Context, address string) int {
	if askCeiling, lowered := ceilings.urlMetadataAskCeilingOfEachPeer[address]; lowered {
		return askCeiling
	}

	return ceilings.mostDocuments
}

func settingsOfOnePartition() spreadSettings {
	return spreadSettings{
		query:                          firstWord + " " + secondWord,
		askablePeers:                   []string{"first", "second"},
		partitions:                     onePartitionOfTheRing,
		documentsToMatchCeiling:        documentsToMatchCeiling,
		urlMetadataAskDocumentsCeiling: urlMetadataAskDocumentsCeiling,
		peersHoldingOneWord:            peersHoldingOneWord,
	}
}

func (settings spreadSettings) spread(
	network *peerNetwork,
	observer wordjoined.WordJoinedSpreadObserver,
) queryanswers.AnsweredQuery {
	ctx := context.Background()
	if settings.queryBudget > 0 {
		var endQuery context.CancelFunc
		ctx, endQuery = context.WithTimeout(ctx, settings.queryBudget)
		defer endQuery()
	}
	query := searchquery.QueryFrom(settings.query, "")
	queryWordDocumentAmounts := settings.queryWordDocumentAmounts
	if queryWordDocumentAmounts == nil {
		queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{})
	}

	return wordjoined.New(
		replicasOf(network),
		network,
		queryWordDocumentAmounts,
		settings.urlMetadataLookupCutoff,
		func(uint) uint { return settings.sampledPartition },
		urlMetadataAskCeilingsOfThePeers{
			mostDocuments:                   settings.urlMetadataAskDocumentsCeiling,
			urlMetadataAskCeilingOfEachPeer: settings.urlMetadataAskCeilingOfEachPeer,
		},
		settings.documentsToMatchCeiling,
		settings.partitions,
		settings.peersHoldingOneWord,
		observer,
	).SpreadOverPeers(
		ctx,
		query,
		settings.choice.ChosenPeersPerQueryWordFor(
			query.HashesOfWordsAndCompoundWordsUpTo(compoundWordsCeiling),
			peersAt(settings.askablePeers),
		),
	)
}

func answeredQueryFrom(
	network *peerNetwork,
	observer wordjoined.WordJoinedSpreadObserver,
) queryanswers.AnsweredQuery {
	return settingsOfOnePartition().spread(network, observer)
}

func spreadOver(
	choice responsiblePeers,
	network *peerNetwork,
	observer wordjoined.WordJoinedSpreadObserver,
) {
	settings := settingsOfOnePartition()
	settings.choice = choice
	settings.spread(network, observer)
}

func distinctDocumentsAskedMetadataFor(asks []peerasks.URLMetadataAsk) []yacymodel.URLHash {
	documentsToAskMetadataFor := map[yacymodel.URLHash]struct{}{}
	for _, ask := range asks {
		for _, document := range ask.Documents {
			documentsToAskMetadataFor[document] = struct{}{}
		}
	}

	return documentsInTheirHashOrder(slices.Collect(maps.Keys(documentsToAskMetadataFor)))
}

func documentsInTheirHashOrder(documents []yacymodel.URLHash) []yacymodel.URLHash {
	documentsInOrder := slices.Clone(documents)
	slices.SortFunc(documentsInOrder, func(first, second yacymodel.URLHash) int {
		return strings.Compare(first.String(), second.String())
	})

	return documentsInOrder
}

func documentHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hashes := documentHashesOf([]string{address})
	if len(hashes) != 1 {
		t.Fatalf("%q has no document hash", address)
	}

	return hashes[0]
}

func foundDocumentsIn(answeredQuery queryanswers.AnsweredQuery) []yacymodel.URLHash {
	foundDocuments := make([]yacymodel.URLHash, 0, len(answeredQuery.FoundDocuments))
	for _, foundDocument := range answeredQuery.FoundDocuments {
		foundDocuments = append(foundDocuments, foundDocument.Hash)
	}

	return documentsInTheirHashOrder(foundDocuments)
}

func TestTheDiscoveryKeepsHalfTheTimeTheQueryHasLeft(t *testing.T) {
	t.Parallel()

	const queryBudget = 3 * time.Second

	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: {"https://shared.example/"}},
		"second": {secondWord: {"https://shared.example/"}},
	})
	settings := settingsOfOnePartition()
	settings.queryBudget = queryBudget

	settings.spread(network, &recordedSpreads{})

	timeLeftAtTheFirstCall := network.timeLeftAtEachCallInTheirOrder[0]
	if timeLeftAtTheFirstCall < queryBudget/2-queryBudget/10 ||
		timeLeftAtTheFirstCall > queryBudget/2+queryBudget/10 {
		t.Fatalf(
			"the discovery kept %s of the %s the query has, want a half",
			timeLeftAtTheFirstCall,
			queryBudget,
		)
	}
	if len(network.urlMetadataAsks) == 0 {
		t.Fatal("the spread asked no peer for metadata, want the joined document looked up")
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

	spreadOver(responsiblePeers{peerAddressesPerWord: map[string][]string{
		firstWord:  {"first"},
		secondWord: {"second"},
	}}, network, observer)

	for _, ask := range network.replicaAsks {
		if ask.Peer.Address == "first" && ask.Word != yacymodel.WordHash(firstWord) {
			t.Fatalf("peer %q was asked what it holds for a word it is not responsible for",
				ask.Peer.Address)
		}
		if ask.Peer.Address == "second" && ask.Word != yacymodel.WordHash(secondWord) {
			t.Fatalf("peer %q was asked what it holds for a word it is not responsible for",
				ask.Peer.Address)
		}
	}
	if len(network.replicaAsks) != 2 {
		t.Fatalf(
			"%d asks were put, want one for each responsible peer",
			len(network.replicaAsks),
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
	if performed.DiscoveryRound.AmountOfQueryWords != 2 ||
		performed.AmountOfJoinedDocuments != 1 {
		t.Fatalf("the spread reported %+v, want two words and one joined document",
			performed)
	}
	if performed.DiscoveryRound.AmountOfQueryWordsHeldByNoPeer != 0 ||
		performed.URLMetadataLookupRound.AmountOfLookedUpDocumentsWithMetadata != 1 {
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

	if observer.performed[0].DiscoveryRound.AmountOfQueryWordsHeldByNoPeer != 1 {
		t.Fatalf(
			"the spread reported %d query words held by no peer, want the one word nobody held",
			observer.performed[0].DiscoveryRound.AmountOfQueryWordsHeldByNoPeer,
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

	settings := settingsOfOnePartition()
	settings.urlMetadataAskDocumentsCeiling = documentsOneURLMetadataAskNames
	settings.spread(network, &recordedSpreads{})

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

func TestAPeerWithALoweredCeilingIsAskedMetadataForAtMostThatManyDocuments(t *testing.T) {
	t.Parallel()

	held := []string{
		"https://first.example/",
		"https://second.example/",
		"https://third.example/",
	}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: held, secondWord: held},
		"second": {firstWord: held, secondWord: held},
	})

	settings := settingsOfOnePartition()
	settings.urlMetadataAskCeilingOfEachPeer = map[string]int{
		"second": documentsOneURLMetadataAskNames,
	}
	settings.spread(network, &recordedSpreads{})

	documentsOfEachPeer := map[string]int{}
	for _, ask := range network.urlMetadataAsks {
		documentsOfEachPeer[ask.Peer.Address] = len(ask.Documents)
	}
	if documentsOfEachPeer["first"] != len(held) ||
		documentsOfEachPeer["second"] != documentsOneURLMetadataAskNames {
		t.Fatalf(
			"the peers were asked metadata for %v documents, want every held one of the first "+
				"and the one its lowered ceiling allows of the second",
			documentsOfEachPeer,
		)
	}
}

func TestAPeerTheCeilingLimitsIsAskedMetadataForTheLeastHeldDocuments(t *testing.T) {
	t.Parallel()

	first := "https://first.example/"
	second := "https://second.example/"

	for _, heldByBothPeers := range []string{first, second} {
		heldByOnePeer := first
		if heldByBothPeers == first {
			heldByOnePeer = second
		}
		settings := settingsOfOnePartition()
		settings.urlMetadataAskCeilingOfEachPeer = map[string]int{
			"first": documentsOneURLMetadataAskNames,
		}

		got := documentsEachPeerIsAskedMetadataFor(settings, heldByBothPeers, heldByOnePeer)

		want := map[string][]yacymodel.URLHash{
			"first":  {documentHashOf(t, heldByOnePeer)},
			"second": {documentHashOf(t, heldByBothPeers)},
		}
		if !maps.EqualFunc(got, want, slices.Equal) {
			t.Fatalf("the peers were asked metadata for %v, want %v", got, want)
		}
	}
}

func TestAPeerTheCeilingDoesNotLimitIsAskedMetadataForEveryDocumentItHolds(t *testing.T) {
	t.Parallel()

	heldByBothPeers := "https://first.example/"
	heldByOnePeer := "https://second.example/"
	settings := settingsOfOnePartition()
	settings.urlMetadataAskCeilingOfEachPeer = map[string]int{"first": 2}

	got := documentsEachPeerIsAskedMetadataFor(settings, heldByBothPeers, heldByOnePeer)

	want := map[string][]yacymodel.URLHash{
		"first": documentsInTheirHashOrder(
			documentHashesOf([]string{heldByBothPeers, heldByOnePeer}),
		),
		"second": {documentHashOf(t, heldByBothPeers)},
	}
	if !maps.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("the peers were asked metadata for %v, want %v", got, want)
	}
}

func documentsEachPeerIsAskedMetadataFor(
	settings spreadSettings,
	heldByBothPeers string,
	heldByOnePeer string,
) map[string][]yacymodel.URLHash {
	joined := []string{heldByBothPeers, heldByOnePeer}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: joined, secondWord: joined},
		"second": {firstWord: {heldByBothPeers}, secondWord: {heldByBothPeers}},
	})
	settings.spread(network, &recordedSpreads{})

	documentsOfEachPeer := map[string][]yacymodel.URLHash{}
	for _, ask := range network.urlMetadataAsks {
		documentsOfEachPeer[ask.Peer.Address] = documentsInTheirHashOrder(ask.Documents)
	}

	return documentsOfEachPeer
}

func TestTheSpreadReportsTheWholeJoinBesideTheDocumentsItAskedMetadataFor(t *testing.T) {
	t.Parallel()

	joined := []string{"https://first.example/", "https://second.example/"}
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: joined, secondWord: joined},
	})
	observer := &recordedSpreads{}

	settings := settingsOfOnePartition()
	settings.urlMetadataAskDocumentsCeiling = documentsOneURLMetadataAskNames
	settings.spread(network, observer)

	performed := observer.performed[0]
	if performed.AmountOfJoinedDocuments != 2 ||
		performed.URLMetadataLookupRound.AmountOfLookedUpDocuments != 1 {
		t.Fatalf(
			"the spread reported %+v, want two joined documents and one asked metadata for",
			performed,
		)
	}
	if performed.URLMetadataLookupRound.AmountOfLookedUpDocumentsWithMetadata != 1 {
		t.Fatalf(
			"the spread reported %d documents back, want the one it asked metadata for",
			performed.URLMetadataLookupRound.AmountOfLookedUpDocumentsWithMetadata,
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

	settings := settingsOfOnePartition()
	settings.askablePeers = []string{"first", "second", "third", "fourth"}
	settings.peersHoldingOneWord = peersOfOneWord
	settings.spread(network, &recordedSpreads{})

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

func TestOnlyTheJoinedDocumentsAreFound(t *testing.T) {
	t.Parallel()

	joined := "https://joined.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {joined, "https://unjoined.example/"}, secondWord: {joined}},
	})

	answeredQuery := answeredQueryFrom(network, &recordedSpreads{})

	if got, wanted := foundDocumentsIn(
		answeredQuery,
	), documentHashesOf([]string{joined}); !slices.Equal(got, wanted) {
		t.Fatalf("the spread found %v, want only the joined document %v", got, wanted)
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

func TestAJoinedDocumentIsFoundThroughItsMetadata(t *testing.T) {
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

func TestTheLookupEndsOnceItsAnswersCoverEveryDocumentWithoutTheStuckPeer(t *testing.T) {
	t.Parallel()

	const answerDelayOfAStuckPeer = 2 * time.Second

	joined := []string{"https://joined.example/"}
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: joined, secondWord: joined},
		"second": {firstWord: joined, secondWord: joined},
	})
	network.metadataDelayOfEachPeer["second"] = answerDelayOfAStuckPeer
	observer := &recordedSpreads{}

	startedAt := time.Now()
	answeredQueryFrom(network, observer)
	timeSpent := time.Since(startedAt)

	performed := observer.performed[0].URLMetadataLookupRound
	if performed.EndReason != wordjoined.URLMetadataLookupEndedByCoverage ||
		performed.AmountOfLookedUpDocumentsWithMetadata != 1 ||
		timeSpent >= answerDelayOfAStuckPeer {
		t.Fatalf(
			"the lookup reported %+v after %v, want it to end by coverage with the document "+
				"before the %v the stuck peer takes",
			performed, timeSpent, answerDelayOfAStuckPeer,
		)
	}
}

func TestTheLookupWaitsForTheStuckPeerWhoseDocumentNoOtherPeerSent(t *testing.T) {
	t.Parallel()

	shared, onlyStuck, onlyFailing := "https://shared.example/",
		"https://only-stuck.example/", "https://only-failing.example/"
	network := networkOf(map[string]map[string][]string{
		"first":   {firstWord: {shared}, secondWord: {shared}},
		"stuck":   {firstWord: {shared, onlyStuck}, secondWord: {shared, onlyStuck}},
		"failing": {firstWord: {onlyFailing}, secondWord: {onlyFailing}},
	})
	network.metadataDelayOfEachPeer["stuck"] = 200 * time.Millisecond
	network.peersFailingTheURLMetadataCall["failing"] = struct{}{}
	settings := settingsOfOnePartition()
	settings.askablePeers = []string{"first", "stuck", "failing"}
	observer := &recordedSpreads{}

	settings.spread(network, observer)

	performed := observer.performed[0].URLMetadataLookupRound
	if performed.EndReason != wordjoined.URLMetadataLookupEndedByEveryAskSettled ||
		performed.AmountOfLookedUpDocumentsWithMetadata != 2 {
		t.Fatalf(
			"the lookup reported %+v, want it to end once every ask settled, "+
				"with the document only the stuck peer sent",
			performed,
		)
	}
}

const (
	answerDelayOfAStuckPeer         = time.Hour
	queryBudgetOfALookupWithACutoff = 10 * time.Second
	shortGrace                      = 50 * time.Millisecond
)

var cutoffAtNinetyPercent = wordjoined.URLMetadataLookupCutoff{
	PercentOfDocuments: 90,
	Grace:              shortGrace,
}

func addressesOfDocuments(site string, amountOfDocuments int) []string {
	addresses := make([]string, 0, amountOfDocuments)
	for number := range amountOfDocuments {
		addresses = append(addresses, fmt.Sprintf("https://%s.example/%d", site, number))
	}

	return addresses
}

func networkOfPeersHoldingBothWords(documentsOfEachPeer map[string][]string) *peerNetwork {
	documentsPerWordPerPeer := map[string]map[string][]string{}
	for address, documents := range documentsOfEachPeer {
		documentsPerWordPerPeer[address] = map[string][]string{
			firstWord: documents, secondWord: documents,
		}
	}

	return networkOf(documentsPerWordPerPeer)
}

type lookupWithACutoff struct {
	performed      wordjoined.PerformedURLMetadataLookupRound
	foundDocuments []yacymodel.URLHash
	timeSpent      time.Duration
}

func lookupOver(
	network *peerNetwork,
	cutoff wordjoined.URLMetadataLookupCutoff,
	queryBudget time.Duration,
) lookupWithACutoff {
	settings := settingsOfOnePartition()
	settings.askablePeers = slices.Sorted(maps.Keys(network.documentsPerWordPerPeer))
	settings.urlMetadataAskDocumentsCeiling = 20
	settings.documentsToMatchCeiling = 20
	settings.queryBudget = queryBudget
	settings.urlMetadataLookupCutoff = cutoff
	observer := &recordedSpreads{}

	startedAt := time.Now()
	answeredQuery := settings.spread(network, observer)

	return lookupWithACutoff{
		performed:      observer.performed[0].URLMetadataLookupRound,
		foundDocuments: foundDocumentsIn(answeredQuery),
		timeSpent:      time.Since(startedAt),
	}
}

func networkWithTenDocumentsOneOnlyAStuckPeerHolds() *peerNetwork {
	network := networkOfPeersHoldingBothWords(map[string][]string{
		"fast":  addressesOfDocuments("fast", 9),
		"stuck": addressesOfDocuments("stuck", 1),
	})
	network.metadataDelayOfEachPeer["stuck"] = answerDelayOfAStuckPeer

	return network
}

func TestTheLookupIsCutOffAGraceAfterMostDocumentsSettled(t *testing.T) {
	t.Parallel()

	network := networkWithTenDocumentsOneOnlyAStuckPeerHolds()

	lookup := lookupOver(network, cutoffAtNinetyPercent, queryBudgetOfALookupWithACutoff)

	if lookup.performed.EndReason != wordjoined.URLMetadataLookupEndedByCutoff ||
		lookup.performed.AmountOfDocumentsCutOffDuringLookup != 1 ||
		lookup.timeSpent < shortGrace || lookup.timeSpent >= queryBudgetOfALookupWithACutoff/2 {
		t.Fatalf(
			"the lookup reported %+v after %v, want it cut off a grace of %v after most "+
				"documents settled, with the document of the stuck peer cut off",
			lookup.performed, lookup.timeSpent, shortGrace,
		)
	}
	if want := documentsInTheirHashOrder(
		documentHashesOf(addressesOfDocuments("fast", 9)),
	); !slices.Equal(lookup.foundDocuments, want) {
		t.Fatalf(
			"the spread found %v, want the nine documents the fast peer sent",
			lookup.foundDocuments,
		)
	}
	if network.urlMetadataLookupContextOfEachPeer["stuck"].Err() == nil {
		t.Fatal("the ask of the stuck peer is still in flight, want it cancelled")
	}
}

func TestTheLookupWithTheCutoffOffWaitsUntilItsRoundEnds(t *testing.T) {
	t.Parallel()

	const queryBudget = 400 * time.Millisecond

	lookup := lookupOver(
		networkWithTenDocumentsOneOnlyAStuckPeerHolds(),
		wordjoined.URLMetadataLookupCutoff{Grace: shortGrace},
		queryBudget,
	)

	if lookup.performed.EndReason != wordjoined.URLMetadataLookupEndedByEveryAskSettled ||
		lookup.performed.AmountOfDocumentsCutOffDuringLookup != 0 ||
		lookup.timeSpent < queryBudget/2 {
		t.Fatalf(
			"the lookup reported %+v after %v, want it to wait for the stuck peer until "+
				"the round of the %v query ends",
			lookup.performed, lookup.timeSpent, queryBudget,
		)
	}
}

func TestADocumentOnlyARefusingPeerHoldsCountsAsSettled(t *testing.T) {
	t.Parallel()

	network := networkOfPeersHoldingBothWords(map[string][]string{
		"fast":     addressesOfDocuments("fast", 16),
		"refusing": addressesOfDocuments("refusing", 3),
		"stuck":    addressesOfDocuments("stuck", 1),
	})
	network.peersFailingTheURLMetadataCall["refusing"] = struct{}{}
	network.metadataDelayOfEachPeer["stuck"] = answerDelayOfAStuckPeer

	lookup := lookupOver(network, cutoffAtNinetyPercent, queryBudgetOfALookupWithACutoff)

	if lookup.performed.EndReason != wordjoined.URLMetadataLookupEndedByCutoff ||
		lookup.performed.AmountOfDocumentsCutOffDuringLookup != 1 ||
		lookup.performed.AmountOfLookedUpDocumentsWithMetadata != 16 {
		t.Fatalf(
			"the lookup reported %+v, want it cut off with the documents of the refusing "+
				"peer settled and only the document of the stuck peer cut off",
			lookup.performed,
		)
	}
}

func TestADocumentAnsweredByOnePeerIsSettledWhileAnotherPeerIsStuck(t *testing.T) {
	t.Parallel()

	fastDocuments := addressesOfDocuments("fast", 9)
	network := networkOfPeersHoldingBothWords(map[string][]string{
		"fast":  fastDocuments,
		"stuck": {fastDocuments[0], "https://only-stuck.example/"},
	})
	network.metadataDelayOfEachPeer["stuck"] = answerDelayOfAStuckPeer

	lookup := lookupOver(network, cutoffAtNinetyPercent, queryBudgetOfALookupWithACutoff)

	if lookup.performed.EndReason != wordjoined.URLMetadataLookupEndedByCutoff ||
		lookup.performed.AmountOfDocumentsCutOffDuringLookup != 1 ||
		lookup.performed.AmountOfLookedUpDocumentsWithMetadata != 9 {
		t.Fatalf(
			"the lookup reported %+v, want the document the fast peer sent settled and "+
				"only the document of the stuck peer cut off",
			lookup.performed,
		)
	}
}

func TestTheLookupEndsByCoverageWhenTheLastDocumentComesBeforeTheGraceEnds(t *testing.T) {
	t.Parallel()

	const longGrace = 5 * time.Second

	fastDocuments := addressesOfDocuments("fast", 9)
	slowDocument := "https://slow.example/"
	network := networkOfPeersHoldingBothWords(map[string][]string{
		"fast":  fastDocuments,
		"slow":  {slowDocument},
		"stuck": {fastDocuments[0], slowDocument},
	})
	network.metadataDelayOfEachPeer["slow"] = 30 * time.Millisecond
	network.metadataDelayOfEachPeer["stuck"] = answerDelayOfAStuckPeer

	lookup := lookupOver(
		network,
		wordjoined.URLMetadataLookupCutoff{PercentOfDocuments: 90, Grace: longGrace},
		queryBudgetOfALookupWithACutoff,
	)

	if lookup.performed.EndReason != wordjoined.URLMetadataLookupEndedByCoverage ||
		lookup.performed.AmountOfDocumentsCutOffDuringLookup != 0 ||
		lookup.timeSpent >= longGrace {
		t.Fatalf(
			"the lookup reported %+v after %v, want it to end by coverage before the %v grace",
			lookup.performed, lookup.timeSpent, longGrace,
		)
	}
}

func TestTheAnswersCarryTheDocumentsTheNetworkHoldsForEachQueryWord(t *testing.T) {
	t.Parallel()

	joined := addressesOfDocuments("joined", 3)
	network := networkOf(map[string]map[string][]string{
		"first":  {firstWord: joined, secondWord: joined},
		"second": {firstWord: joined, secondWord: joined},
	})

	answers := answeredQueryFrom(network, &recordedSpreads{})

	want := documentsHeldForBothQueryWords(len(joined))
	if got := answers.DocumentsHeldPerQueryWord; !maps.Equal(got, want) {
		t.Fatalf("the answers carry %v documents per query word, want %v", got, want)
	}
}

func TestEveryPartitionOfAQueryWordAddsWhatTheAbstractsOfItsReplicasList(t *testing.T) {
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

func TestAPartitionOfAnEvenAmountOfAbstractsTakesTheLowerMiddleOne(t *testing.T) {
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

func TestAPartitionNoReplicaAnsweredTakesTheMiddleOfTheAnsweredPartitions(t *testing.T) {
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

func TestAQueryWordNoReplicaAnsweredCarriesNoDocumentsHeld(t *testing.T) {
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
	silentPeers map[string]struct{},
	partitions yacymodel.DHTRingPartitions,
	choice responsiblePeers,
) map[yacymodel.Hash]int {
	documentsOfEachPeer := map[string][]string{}
	for address, documentsHeld := range documentsHeldByEachPeer {
		documentsOfEachPeer[address] = addressesOfDocuments(address, documentsHeld)
	}
	network := networkOfPeersHoldingBothWords(documentsOfEachPeer)
	network.silentPeers = silentPeers

	settings := settingsOfOnePartition()
	settings.choice = choice
	settings.partitions = partitions
	settings.documentsToMatchCeiling = 0
	settings.askablePeers = addressesAcross(documentsHeldByEachPeer, silentPeers)

	return settings.spread(network, &recordedSpreads{}).DocumentsHeldPerQueryWord
}

func addressesAcross(
	documentsHeldByEachPeer map[string]int,
	silentPeers map[string]struct{},
) []string {
	addresses := make([]string, 0, len(documentsHeldByEachPeer)+len(silentPeers))
	for address := range documentsHeldByEachPeer {
		addresses = append(addresses, address)
	}
	for address := range silentPeers {
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

func TestTheSpreadReportsTheAmountOfDocumentsInEachAbstract(t *testing.T) {
	t.Parallel()

	held := addressesOfDocuments("held", 2)
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: held, secondWord: held[:1]},
	})
	observer := &recordedSpreads{}

	spreadOver(responsiblePeers{peerAddressesPerWord: map[string][]string{
		firstWord:  {"first"},
		secondWord: {"first"},
	}}, network, observer)

	if got := observer.performed[0].DiscoveryRound.AmountOfDocumentsInEachAbstract; !slices.Equal(
		slices.Sorted(slices.Values(got)), []int{1, 2},
	) {
		t.Fatalf("the spread reported %v documents in each abstract, want 1 and 2", got)
	}
}

func TestEachAskOfAQueryWordNamesThePartitionOfItsReplica(t *testing.T) {
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
	if got := replicasAskedForTheWord(
		yacymodel.WordHash(firstWord), network.replicaAsks,
	); !slices.Equal(got, wanted) {
		t.Fatalf("the spread asked %v for the word, want %v", got, wanted)
	}
}

func spreadAcrossPartitions(
	network *peerNetwork,
	choice responsiblePeers,
	partitions yacymodel.DHTRingPartitions,
) {
	settings := settingsOfOnePartition()
	settings.choice = choice
	settings.partitions = partitions
	settings.askablePeers = []string{"nearest-of-first", "next-of-first", "nearest-of-second"}
	settings.spread(network, &recordedSpreads{})
}

func replicasAskedForTheWord(
	word yacymodel.Hash,
	asks []replicaAsk,
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

	return slices.Sorted(slices.Values(replicasAsked))
}

func TestAFoundDocumentHoldsNoAmountOfLinks(t *testing.T) {
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

func TestADocumentInTheAbstractOfTheCompoundWordOfTwoWordsIsJoinedForBoth(t *testing.T) {
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
			"the spread found %+v, want the one document in the abstract of the compound word",
			answers.FoundDocuments,
		)
	}
}

func networkWhereTheSecondWordLeadsFromTheSampleInPartitionZero(t *testing.T) *peerNetwork {
	t.Helper()

	documentsInPartitionZero := addressesInPartition(t, twoPartitionsOfTheRing, 0, 3)
	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 3)

	return networkOf(map[string]map[string][]string{
		"first-in-0":  {firstWord: documentsInPartitionZero},
		"first-in-1":  {firstWord: documentsInPartitionOne},
		"second-in-0": {secondWord: documentsInPartitionZero[:1]},
		"second-in-1": {secondWord: documentsInPartitionOne},
	})
}

func TestAQueryWhoseWordsAllHaveAnAmountLeadsWithTheRarestAndTakesNoSample(t *testing.T) {
	t.Parallel()

	network := networkWhereTheSecondWordLeadsFromTheSampleInPartitionZero(t)
	settings := settingsOfTwoPartitions()
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(
		map[string]int{firstWord: 1, secondWord: 5},
	)
	observer := &recordedSpreads{}

	settings.spread(network, observer)

	if choice := observer.performed[0].DiscoveryRound.LeadingQueryWordChoice; choice !=
		wordjoined.RarestQueryWordRemembered {
		t.Fatalf("the spread chose the leading word by %q, want the remembered amounts", choice)
	}
	asksOfTheOtherWord := asksOfTheWord(secondWord, network.replicaAsks)
	if got := asksNamingDocumentsToMatchAmong(asksOfTheOtherWord); len(got) !=
		len(asksOfTheOtherWord) || !slices.Equal(partitionsAskedAmong(got), []uint{0, 1}) {
		t.Fatalf(
			"the spread asked the other word %v, want it asked for the documents to match "+
				"in every partition and never whole in a sample",
			asksOfTheOtherWord,
		)
	}
	if got := asksNamingDocumentsToMatchAmong(
		asksOfTheWord(firstWord, network.replicaAsks),
	); len(got) != 0 {
		t.Fatalf("the spread asked the leading word %v naming documents, want it asked whole", got)
	}
}

func TestAQueryWithAWordWithoutAnAmountTakesTheSample(t *testing.T) {
	t.Parallel()

	network := networkWhereTheSecondWordLeadsFromTheSampleInPartitionZero(t)
	settings := settingsOfTwoPartitions()
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{firstWord: 1})
	observer := &recordedSpreads{}

	settings.spread(network, observer)

	if choice := observer.performed[0].DiscoveryRound.LeadingQueryWordChoice; choice !=
		wordjoined.RarestQueryWordWithASample {
		t.Fatalf("the spread chose the leading word by %q, want a sample", choice)
	}
	if got := asksNamingDocumentsToMatchAmong(
		asksOfTheWord(secondWord, network.replicaAsks),
	); len(got) != 0 {
		t.Fatalf("the spread asked %v naming documents, want the rarest sampled word leading", got)
	}
}

func TestTheSpreadRemembersTheDocumentsHeldAcrossTheRingForEachWordAReplicaListed(t *testing.T) {
	t.Parallel()

	network := networkWhereTheSecondWordLeadsFromTheSampleInPartitionZero(t)
	settings := settingsOfTwoPartitions()
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{})

	settings.spread(network, &recordedSpreads{})

	want := map[yacymodel.Hash]int{
		yacymodel.WordHash(firstWord): 3 + 3, yacymodel.WordHash(secondWord): 1 + 3,
	}
	if got := settings.queryWordDocumentAmounts.amountOfEachWord; !maps.Equal(got, want) {
		t.Fatalf("the spread remembered %v, want %v", got, want)
	}
}

func TestAnAbstractOfAnAskNamingTheDocumentsToMatchSaysNothingOfWhatTheWordHolds(
	t *testing.T,
) {
	t.Parallel()

	documentsInPartitionZero := addressesInPartition(t, twoPartitionsOfTheRing, 0, 3)
	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 5)
	network := networkOf(map[string]map[string][]string{
		"first-in-0":  {firstWord: documentsInPartitionZero},
		"first-in-1":  {firstWord: documentsInPartitionOne},
		"second-in-0": {secondWord: documentsInPartitionZero[:1]},
		"second-in-1": {secondWord: documentsInPartitionOne[:1]},
	})
	settings := settingsOfTwoPartitions()
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{})

	settings.spread(network, &recordedSpreads{})

	want := map[yacymodel.Hash]int{
		yacymodel.WordHash(firstWord): 3 + 3, yacymodel.WordHash(secondWord): 1 + 1,
	}
	if got := settings.queryWordDocumentAmounts.amountOfEachWord; !maps.Equal(got, want) {
		t.Fatalf(
			"the spread remembered %v, want %v, with the partition asked for the documents "+
				"to match taking the middle of the others",
			got, want,
		)
	}
}

func TestAWordNoReplicaAnsweredIsNotRemembered(t *testing.T) {
	t.Parallel()

	network := networkWhereTheSecondWordLeadsFromTheSampleInPartitionZero(t)
	network.silentPeers = map[string]struct{}{"first-in-0": {}, "first-in-1": {}}
	settings := settingsOfTwoPartitions()
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{})

	settings.spread(network, &recordedSpreads{})

	if _, remembered := settings.queryWordDocumentAmounts.amountOfEachWord[yacymodel.WordHash(
		firstWord,
	)]; remembered {
		t.Fatal("the spread remembered an amount for a word no replica answered")
	}
}
