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
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/matchingwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
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
	answeredItemsPerWordPerPeer           map[string]map[string][]string
	countsAWordWithEachItem               bool
	documentsHeldForEveryWord             int
	documentsHeldByEachPeer               map[string]int
	documentsPerAnswerOfEachPeer          map[string]int
	peersCountingNoDocument               map[string]struct{}
	peersThatSearched                     map[string]struct{}
	searchDocumentsAsks                   []replicaAsk
	settledWordPartitionsReadAtEachAsk    []int
	urlMetadataAsks                       []peerasks.URLMetadataAsk
	silentPeers                           map[string]struct{}
	peersListingDocumentsTheAskDidNotName map[string]struct{}
	timeLeftAtEachCallInTheirOrder        []time.Duration
}

func networkOf(documentsPerWordPerPeer map[string]map[string][]string) *peerNetwork {
	return &peerNetwork{
		documentsPerWordPerPeer:               documentsPerWordPerPeer,
		peersCountingNoDocument:               map[string]struct{}{},
		peersThatSearched:                     map[string]struct{}{},
		silentPeers:                           map[string]struct{}{},
		peersListingDocumentsTheAskDidNotName: map[string]struct{}{},
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

	network.searchDocumentsAsks = append(network.searchDocumentsAsks, ask)
	network.settledWordPartitionsReadAtEachAsk = append(
		network.settledWordPartitionsReadAtEachAsk, settledAsksRead,
	)
	network.recordTimeLeftIn(ctx)
	if _, silent := network.silentPeers[ask.Peer.Address]; silent {
		return wordpartitionasks.ReplicaAnswer{}, false
	}

	return wordpartitionasks.ReplicaAnswer{
		Replica: ask.Peer,
		ListedDocuments: listedDocumentsFrom(
			network.abstractFor(ask),
			network.matchedDocumentsOf(network.namedAmong(
				ask,
				documentsPerWordOf(network.answeredItemsPerWordPerPeer, ask.Peer.Address, ask.Word),
			)),
		),
		AmountOfDocumentsHeld: network.documentsCountedBy(ask.Peer.Address, ask.Word),
		Searched:              network.searchedBy(ask.Peer.Address),
	}, true
}

func listedDocumentsFrom(
	abstract []yacymodel.URLHash,
	matchedDocuments []wordpartitionasks.ListedDocument,
) []wordpartitionasks.ListedDocument {
	listedDocuments := make([]wordpartitionasks.ListedDocument, 0, len(abstract))
	for _, document := range abstract {
		listedDocuments = append(listedDocuments, wordpartitionasks.ListedDocument{Hash: document})
	}
	for _, matchedDocument := range matchedDocuments {
		place := slices.IndexFunc(
			listedDocuments,
			func(listedDocument wordpartitionasks.ListedDocument) bool {
				return listedDocument.Hash == matchedDocument.Hash
			},
		)
		if place < 0 {
			listedDocuments = append(listedDocuments, matchedDocument)

			continue
		}
		listedDocuments[place] = matchedDocument
	}

	return listedDocuments
}

func (network *peerNetwork) searchedBy(address string) bool {
	_, searched := network.peersThatSearched[address]

	return searched
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
	if len(ask.DocumentsToMatch) > 0 {
		return network.namedAmong(ask, documents)
	}
	documentsPerAnswer, answersInPart := network.documentsPerAnswerOfEachPeer[ask.Peer.Address]
	if !answersInPart || len(documents) <= documentsPerAnswer {
		return documents
	}

	return documents[:documentsPerAnswer]
}

func (network *peerNetwork) namedAmong(
	ask replicaAsk,
	documents []yacymodel.URLHash,
) []yacymodel.URLHash {
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

func (network *peerNetwork) matchedDocumentsOf(
	documents []yacymodel.URLHash,
) []wordpartitionasks.ListedDocument {
	matchedDocuments := make([]wordpartitionasks.ListedDocument, 0, len(documents))
	for _, document := range documents {
		matchedDocument := wordpartitionasks.ListedDocument{
			Hash:     document,
			Metadata: yacymodel.Some(yacymodel.URLMetadata{Hash: document}),
		}
		if network.countsAWordWithEachItem {
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

func (network *peerNetwork) documentsCountedBy(
	address string,
	word yacymodel.Hash,
) yacymodel.Optional[int] {
	if _, countsNoDocument := network.peersCountingNoDocument[address]; countsNoDocument {
		return yacymodel.None[int]()
	}
	if _, answersInPart := network.documentsPerAnswerOfEachPeer[address]; answersInPart {
		return yacymodel.Some(
			len(documentsPerWordOf(network.documentsPerWordPerPeer, address, word)),
		)
	}
	if documentsHeld, countsItsOwn := network.documentsHeldByEachPeer[address]; countsItsOwn {
		return yacymodel.Some(documentsHeld)
	}

	return yacymodel.Some(network.documentsHeldForEveryWord)
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
	for _, ask := range asks {
		outcomesAsTheySettle <- peerasks.URLMetadataAskOutcome{
			Ask: ask,
			Put: true,
			Answer: yacymodel.Some(peerasks.AnsweredURLMetadataAsk{
				Ask:                    ask,
				MetadataOfEachDocument: metadataOfEachDocument(ask.Documents),
			}),
		}
	}
	close(outcomesAsTheySettle)

	return outcomesAsTheySettle
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
	countedPartition                uint
	documentsToMatchCeiling         int
	urlMetadataAskDocumentsCeiling  int
	urlMetadataAskCeilingOfEachPeer map[string]int
	peersHoldingOneWord             int
	queryBudget                     time.Duration
	queryWordDocumentAmounts        *rememberedQueryWordDocumentAmounts
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

type clockThatNeverFires struct{}

func (clockThatNeverFires) After(time.Duration, func()) func() {
	return func() {}
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
) queryfindings.Findings {
	ctx := context.Background()
	if settings.queryBudget > 0 {
		var endQuery context.CancelFunc
		ctx, endQuery = context.WithTimeout(ctx, settings.queryBudget)
		defer endQuery()
	}
	query := queryreading.QueryFrom(settings.query, "")
	queryWordDocumentAmounts := settings.queryWordDocumentAmounts
	if queryWordDocumentAmounts == nil {
		queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{})
	}

	return wordjoined.New(
		replicasOf(network),
		queryWordDocumentAmounts,
		leadingword.New(documentamounts.NewFromCache(
			queryWordDocumentAmounts,
			documentamounts.NewFromReplicas(
				settings.partitions,
				func(uint) uint { return settings.countedPartition },
				documentamounts.FromReplicasObservers{},
			),
			documentamounts.FromCacheObservers{},
		)),
		matchingwords.New(settings.partitions, settings.documentsToMatchCeiling),
		urlmetadataasks.New(
			network,
			urlMetadataAskCeilingsOfThePeers{
				mostDocuments:                   settings.urlMetadataAskDocumentsCeiling,
				urlMetadataAskCeilingOfEachPeer: settings.urlMetadataAskCeilingOfEachPeer,
			},
			urlmetadataasks.Cutoff{},
			clockThatNeverFires{},
			settings.peersHoldingOneWord,
		),
		settings.partitions,
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

func findingsFrom(
	network *peerNetwork,
	observer wordjoined.WordJoinedSpreadObserver,
) queryfindings.Findings {
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

func foundDocumentsIn(findings queryfindings.Findings) []yacymodel.URLHash {
	foundDocuments := make([]yacymodel.URLHash, 0, len(findings.FoundDocuments))
	for _, foundDocument := range findings.FoundDocuments {
		foundDocuments = append(foundDocuments, foundDocument.Hash)
	}

	return documentsInTheirHashOrder(foundDocuments)
}

func TestTheWordAsksKeepHalfTheTimeTheQueryHasLeft(t *testing.T) {
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
			"the word asks kept %s of the %s the query has, want a half",
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

	findingsFrom(network, &recordedSpreads{})

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

	findingsFrom(network, &recordedSpreads{})

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

	findingsFrom(network, &recordedSpreads{})

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

	foundDocuments := findingsFrom(network, &recordedSpreads{}).FoundDocuments

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

	for _, ask := range network.searchDocumentsAsks {
		if ask.Peer.Address == "first" && ask.Word != yacymodel.WordHash(firstWord) {
			t.Fatalf("peer %q was asked what it holds for a word it is not responsible for",
				ask.Peer.Address)
		}
		if ask.Peer.Address == "second" && ask.Word != yacymodel.WordHash(secondWord) {
			t.Fatalf("peer %q was asked what it holds for a word it is not responsible for",
				ask.Peer.Address)
		}
	}
	if len(network.searchDocumentsAsks) != 2 {
		t.Fatalf(
			"%d asks were put, want one for each responsible peer",
			len(network.searchDocumentsAsks),
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

	findingsFrom(network, observer)

	if len(observer.performed) != 1 {
		t.Fatalf("the observer saw %d spreads, want one", len(observer.performed))
	}
	performed := observer.performed[0]
	if performed.AmountOfQueryWords != 2 ||
		performed.AmountOfJoinedDocuments != 1 {
		t.Fatalf("the spread reported %+v, want two words and one joined document",
			performed)
	}
	if performed.AmountOfQueryWordsHeldByNoPeer != 0 ||
		performed.URLMetadataAsks.AmountOfAskedDocumentsWithMetadata != 1 {
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

	findingsFrom(network, observer)

	if observer.performed[0].AmountOfQueryWordsHeldByNoPeer != 1 {
		t.Fatalf(
			"the spread reported %d query words held by no peer, want the one word nobody held",
			observer.performed[0].AmountOfQueryWordsHeldByNoPeer,
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

	findingsFrom(network, observer)

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
		performed.URLMetadataAsks.AmountOfAskedDocuments != 1 {
		t.Fatalf(
			"the spread reported %+v, want two joined documents and one asked metadata for",
			performed,
		)
	}
	if performed.URLMetadataAsks.AmountOfAskedDocumentsWithMetadata != 1 {
		t.Fatalf(
			"the spread reported %d documents back, want the one it asked metadata for",
			performed.URLMetadataAsks.AmountOfAskedDocumentsWithMetadata,
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

	findingsFrom(network, &recordedSpreads{})

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

	foundDocuments := findingsFrom(network, &recordedSpreads{}).FoundDocuments

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

func TestTheFindingsCarryTheWordsOfTheQuery(t *testing.T) {
	t.Parallel()

	answered := "https://answered.example/"
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: {answered}, secondWord: {answered}},
	})

	findings := findingsFrom(network, &recordedSpreads{})

	want := []yacymodel.Hash{yacymodel.WordHash(firstWord), yacymodel.WordHash(secondWord)}
	if !slices.Equal(findings.QueryWords, want) {
		t.Fatalf("the findings carry the query words %v, want %v", findings.QueryWords, want)
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

	findings := findingsFrom(network, &recordedSpreads{})
	foundDocuments := findings.FoundDocuments

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

	findings := findingsFrom(network, &recordedSpreads{})
	foundDocuments := findings.FoundDocuments

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

	foundDocuments := findingsFrom(network, &recordedSpreads{}).FoundDocuments

	if len(foundDocuments) != 1 || foundDocuments[0].Hash != documentHashOf(t, joined) {
		t.Fatalf("the spread found %v, want the joined document once", foundDocuments)
	}
}

func TestTheFindingsCarryTheDocumentsTheNetworkHoldsForEachQueryWord(t *testing.T) {
	t.Parallel()

	joined := []string{"https://joined.example/"}
	network := networkOf(map[string]map[string][]string{
		"first": {firstWord: joined, secondWord: joined},
	})
	network.documentsHeldForEveryWord = 512

	findings := findingsFrom(network, &recordedSpreads{})

	want := documentsHeldForBothQueryWords(512)
	if got := findings.DocumentsHeldPerQueryWord; !maps.Equal(got, want) {
		t.Fatalf("the findings carry %v documents per query word, want %v", got, want)
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

	findings := findingsFrom(network, &recordedSpreads{})

	want := documentsHeldForBothQueryWords(512)
	if got := findings.DocumentsHeldPerQueryWord; !maps.Equal(got, want) {
		t.Fatalf("the findings carry %v documents per query word, want %v", got, want)
	}
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

	spreadOver(responsiblePeers{peerAddressesPerWord: map[string][]string{
		firstWord:  {"first"},
		secondWord: {"first"},
	}}, network, observer)

	performed := observer.performed[0]
	wordAsks := performed.WordAsks
	if performed.AmountOfJoinedDocumentsWithMetadata != 1 ||
		wordAsks.AmountOfMatchedDocumentsAcrossAnswers != 2 ||
		wordAsks.AmountOfMatchedDocumentsWithAPosting != 2 {
		t.Fatalf(
			"the spread reported %+v, want the joined document answered once and two counts",
			performed,
		)
	}
	if !slices.Equal(
		wordAsks.AmountOfDocumentsHeldInEachAnswer, []int{512, 512},
	) {
		t.Fatalf(
			"the spread reported %v documents held per query word, want 512 for each answer",
			wordAsks.AmountOfDocumentsHeldInEachAnswer,
		)
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
		yacymodel.WordHash(firstWord), network.searchDocumentsAsks,
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

	findings := findingsFrom(network, &recordedSpreads{})
	foundDocuments := findings.FoundDocuments

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

	findings := findingsFrom(network, &recordedSpreads{})
	foundDocuments := findings.FoundDocuments

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

	findings := findingsFrom(network, &recordedSpreads{})

	if len(findings.FoundDocuments) != 1 ||
		findings.FoundDocuments[0].Hash != documentHashesOf([]string{compounded})[0] {
		t.Fatalf(
			"the spread found %+v, want the one document in the abstract of the compound word",
			findings.FoundDocuments,
		)
	}
}

func networkWhereTheSecondWordLeadsInPartitionZero(t *testing.T) *peerNetwork {
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

func TestAQueryWhoseWordsAllHaveACachedAmountLeadsWithTheRarestCachedWord(t *testing.T) {
	t.Parallel()

	network := networkWhereTheSecondWordLeadsInPartitionZero(t)
	settings := settingsOfTwoPartitions()
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(
		map[string]int{firstWord: 1, secondWord: 5},
	)

	settings.spread(network, &recordedSpreads{})

	asksOfTheMatchingWord := asksOfTheWord(secondWord, network.searchDocumentsAsks)
	if got := asksNamingDocumentsToMatchAmong(asksOfTheMatchingWord); len(got) !=
		len(asksOfTheMatchingWord) || !slices.Equal(partitionsAskedAmong(got), []uint{0, 1}) {
		t.Fatalf(
			"the spread asked the matching word %v, want it asked for the documents to match "+
				"in every partition and never whole",
			asksOfTheMatchingWord,
		)
	}
	if got := asksNamingDocumentsToMatchAmong(
		asksOfTheWord(firstWord, network.searchDocumentsAsks),
	); len(got) != 0 {
		t.Fatalf("the spread asked the leading word %v naming documents, want it asked whole", got)
	}
}

func TestAQueryWithAWordWithoutACachedAmountLeadsWithTheRarestCountedWord(t *testing.T) {
	t.Parallel()

	network := networkWhereTheSecondWordLeadsInPartitionZero(t)
	settings := settingsOfTwoPartitions()
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{firstWord: 1})

	settings.spread(network, &recordedSpreads{})

	if got := asksNamingDocumentsToMatchAmong(
		asksOfTheWord(secondWord, network.searchDocumentsAsks),
	); len(got) != 0 {
		t.Fatalf("the spread asked %v naming documents, want the rarest counted word leading", got)
	}
}

func TestTheSpreadRemembersTheDocumentsInAPartitionForEachWordAPeerCounted(t *testing.T) {
	t.Parallel()

	network := networkWhereTheSecondWordLeadsInPartitionZero(t)
	network.documentsHeldByEachPeer = map[string]int{
		"first-in-0": 10, "first-in-1": 30, "second-in-0": 4, "second-in-1": 6,
	}
	settings := settingsOfTwoPartitions()
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{})

	settings.spread(network, &recordedSpreads{})

	want := map[yacymodel.Hash]int{
		yacymodel.WordHash(firstWord): 10, yacymodel.WordHash(secondWord): 4,
	}
	if got := settings.queryWordDocumentAmounts.amountOfEachWord; !maps.Equal(got, want) {
		t.Fatalf("the spread remembered %v, want %v", got, want)
	}
}

func TestAWordNoPeerCountedIsNotRemembered(t *testing.T) {
	t.Parallel()

	network := networkWhereTheSecondWordLeadsInPartitionZero(t)
	network.peersCountingNoDocument = map[string]struct{}{"first-in-0": {}, "first-in-1": {}}
	settings := settingsOfTwoPartitions()
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{})

	settings.spread(network, &recordedSpreads{})

	if _, remembered := settings.queryWordDocumentAmounts.amountOfEachWord[yacymodel.WordHash(
		firstWord,
	)]; remembered {
		t.Fatal("the spread remembered an amount for a word no peer counted")
	}
}
