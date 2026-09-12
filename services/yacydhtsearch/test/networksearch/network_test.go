package networksearch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/peerorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	networkName         = "freeworld"
	responseLimit       = 1 << 20
	peersHoldingOneWord = 4
	peerCallsInFlight   = 48
	peerCallBudget      = 3 * time.Second
	queryBudget         = 5 * time.Second
	pageReadBudget      = 3 * time.Second
	peerResults         = 10
	directoryLimit      = 16
	recordCeiling       = 50
	pagesReadPerQuery   = 50
	cooldown            = 5 * time.Second
)

type silentDirectoryObserver struct{}

func (silentDirectoryObserver) PeerAdmitted(context.Context, yacymodel.Hash, int)     {}
func (silentDirectoryObserver) PeerAnswering(context.Context, yacymodel.Hash, string) {}
func (silentDirectoryObserver) PeerSilent(context.Context, yacymodel.Hash)            {}
func (silentDirectoryObserver) PeerDropped(context.Context, yacymodel.Hash)           {}
func (silentDirectoryObserver) DirectoryHolds(context.Context, int, int, int)         {}

type silentOutcome struct{}

func (silentOutcome) PeerAnsweredMatchedItems(context.Context, string, int, time.Duration)  {}
func (silentOutcome) PeerAnsweredURLMetadata(context.Context, string, int, time.Duration)   {}
func (silentOutcome) PeerAnsweredHeldDocuments(context.Context, string, int, time.Duration) {}

func (silentOutcome) PeerRefused(
	context.Context, string, peerasks.AskedFor, int, time.Duration,
) {
}

func (silentOutcome) PeerUnreachable(
	context.Context, string, peerasks.AskedFor, error, time.Duration,
) {
}

func (silentOutcome) PeerAnswerUnreadable(
	context.Context, string, peerasks.AskedFor, error, time.Duration,
) {
}

type recordedQuery struct {
	performed networksearch.PerformedNetworkSearch
}

func (r *recordedQuery) NetworkSearchPerformed(
	_ context.Context,
	search networksearch.PerformedNetworkSearch,
) {
	r.performed = search
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

func peerHolding(t *testing.T, addresses ...string) string {
	t.Helper()

	resources := make([]yacyproto.SearchResource, 0, len(addresses))
	for _, address := range addresses {
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			t.Fatalf("URLHashOf(%q): %v", address, err)
		}
		resources = append(resources, yacyproto.SearchResource{
			Metadata: yacymodel.URLMetadata{Hash: hash, Address: address, Title: "Weather"},
		})
	}
	body := yacyproto.SearchResponse{
		Count:     len(resources),
		Resources: resources,
	}.Encode().Encode()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(body))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func directoryAnsweringAt(t *testing.T, addresses ...string) *peerdirectory.Directory {
	t.Helper()

	directory := peerdirectory.New(
		directoryLimit,
		cooldown,
		time.Now,
		stalestFirst{},
		silentDirectoryObserver{},
	)
	port, err := yacymodel.ParsePort("8090")
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	host, err := yacymodel.ParseHost("10.0.0.1")
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	for index, address := range addresses {
		peer := yacymodel.WordHash(string(rune('a' + index)))
		directory.Admit(t.Context(), []yacymodel.Seed{{
			Hash:           peer,
			PrimaryAddress: yacymodel.Some(host),
			Port:           yacymodel.Some(port),
		}})
		directory.ConfirmAnswering(t.Context(), peer, address)
	}

	return directory
}

type stalestFirst struct{}

func (stalestFirst) StalestPeers(known []peerdirectory.KnownPeer, _ int) []yacymodel.Hash {
	return []yacymodel.Hash{known[0].Hash}
}

func networkOver(
	t *testing.T,
	directory *peerdirectory.Directory,
	observer networksearch.NetworkSearchObserver,
) networksearch.Network {
	t.Helper()

	return networkSearching(t, directory, observer, peerMatchedSpread(t))
}

func peerMatchedSpread(t *testing.T) peermatched.Spread {
	t.Helper()

	return peermatched.New(
		peercallwire.New(
			http.DefaultClient,
			peercallwire.SearchedNetwork{Name: networkName, RingPartitions: ringPartitions(t)},
			peercallwire.PeerCallLimits{
				MaxResponseBytes:  responseLimit,
				PeerCallsInFlight: peerCallsInFlight,
				PeerCallBudget:    peerCallBudget,
			},
			silentOutcome{},
		),
		everyAskablePeer{},
		peerResults,
		peersHoldingOneWord,
		peermatched.PeerMatchedSpreadObservers{},
	)
}

func ringPartitions(t *testing.T) yacymodel.DHTRingPartitions {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("partitions from exponent: %v", err)
	}

	return partitions
}

func networkSearching(
	t *testing.T,
	directory *peerdirectory.Directory,
	observer networksearch.NetworkSearchObserver,
	querySpread networksearch.QuerySpread,
) networksearch.Network {
	t.Helper()

	return networkOrdering(t, directory, observer, querySpread, peerorder.Ordering{})
}

func networkOrdering(
	t *testing.T,
	directory *peerdirectory.Directory,
	observer networksearch.NetworkSearchObserver,
	querySpread networksearch.QuerySpread,
	itemsOrdering networksearch.ItemsOrdering,
) networksearch.Network {
	t.Helper()

	return networksearch.New(
		directory,
		querySpread,
		pagesThatNoOneReads{},
		itemsOrdering,
		queryBudget,
		pageReadBudget,
		pagesReadPerQuery,
		recordCeiling,
		networksearch.NetworkSearchObservers{observer},
	)
}

type pagesThatNoOneReads struct{}

func (pagesThatNoOneReads) DocumentTextPerDocument(
	_ context.Context,
	_ []yacymodel.Hash,
	_ []pagereading.PageToRead,
) map[yacymodel.URLHash]documenttext.DocumentText {
	return nil
}

func TestOneQueryCarriesBackWhatThePeersHold(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t, peerHolding(t, "https://a.example/"))
	network := networkOver(t, directory, observer)

	ranking, outcome := network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if outcome != networksearch.PeersAsked {
		t.Fatalf("Search reached outcome %v, want peers asked", outcome)
	}
	if len(ranking.Items) != 1 || ranking.Items[0].Address != "https://a.example/" {
		t.Fatalf("Search = %+v, want the address the peer holds", ranking.Items)
	}
	if observer.performed.AmountOfAskablePeers != 1 ||
		observer.performed.AmountOfItemsInRanking != 1 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want one peer asked, answered and one item",
			observer.performed,
		)
	}
}

func TestARankingStopsAtTheRecordCeiling(t *testing.T) {
	t.Parallel()

	addresses := make([]string, 0, recordCeiling+1)
	for index := range recordCeiling + 1 {
		addresses = append(addresses, "https://a.example/"+strconv.Itoa(index))
	}
	directory := directoryAnsweringAt(t, peerHolding(t, addresses...))
	network := networkOver(t, directory, &recordedQuery{})

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if len(ranking.Items) != recordCeiling {
		t.Fatalf("Search carried %d items, want the ceiling %d", len(ranking.Items), recordCeiling)
	}
}

func TestAQueryThatReachesNoPeerCarriesBackThatOutcome(t *testing.T) {
	t.Parallel()

	network := networkOver(t, directoryAnsweringAt(t), &recordedQuery{})

	ranking, outcome := network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if len(ranking.Items) != 0 || outcome != networksearch.NoPeerToAsk {
		t.Fatalf(
			"Search = %+v with outcome %v, want an empty ranking and no peer to ask",
			ranking.Items,
			outcome,
		)
	}
}

func TestAQueryWithoutAnIndexedTermReachesNoPeer(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t, peerHolding(t, "https://a.example/"))
	network := networkOver(t, directory, observer)

	ranking, outcome := network.Search(t.Context(), searchquery.QueryFrom("1"))

	if len(ranking.Items) != 0 || observer.performed.AmountOfAskablePeers != 0 {
		t.Fatalf(
			"Search = %+v after asking %d peers, want an empty ranking and no peer asked",
			ranking.Items,
			observer.performed.AmountOfAskablePeers,
		)
	}
	if outcome != networksearch.NoIndexedTermInQuery {
		t.Fatalf("Search reached outcome %v, want no indexed term in the query", outcome)
	}
}

func TestAnAddressTwoPeersHoldIsRankedOnce(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t,
		peerHolding(t, "https://a.example/", "https://shared.example/"),
		peerHolding(t, "https://shared.example/", "https://b.example/"),
	)
	network := networkOver(t, directory, observer)

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if len(ranking.Items) != 3 {
		t.Fatalf("Search = %+v, want the three addresses the two peers hold", ranking.Items)
	}
	if observer.performed.AmountOfItemsAcrossAnswers != 4 ||
		observer.performed.AmountOfItemsInRanking != 3 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want four items answered and three ranked",
			observer.performed,
		)
	}
}

func TestTheRankingReportsHowMuchOfItOnePeerSupplied(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t,
		peerHolding(t, "https://a.example/"),
		peerHolding(t, "https://b.example/"),
	)
	network := networkOver(t, directory, observer)

	network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if observer.performed.AmountOfItemsInRanking != 2 ||
		observer.performed.AmountOfRankedItemsOfTheOnePeer != 1 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want two ranked items and one from the leading peer",
			observer.performed,
		)
	}
}

func TestAPeerThatRepliedHoldingNothingAnswersButSendsNoItem(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t,
		peerHolding(t),
		peerHolding(t, "https://a.example/"),
	)
	network := networkOver(t, directory, observer)

	network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if observer.performed.AmountOfAskablePeers != 2 ||
		observer.performed.AmountOfItemsAcrossAnswers != 1 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want two chosen peers and one item",
			observer.performed,
		)
	}
}

func TestOnePeerCanSupplyTheWholeRanking(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t, peerHolding(t, "https://a.example/", "https://a.example/"))
	network := networkSearching(t, directory, observer, peerMatchedSpread(t))

	network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if observer.performed.AmountOfAskablePeers != 1 ||
		observer.performed.AmountOfItemsInRanking != 1 ||
		observer.performed.AmountOfRankedItemsOfTheOnePeer != 1 ||
		observer.performed.AmountOfItemsAcrossAnswers != 2 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want one askable peer supplying the whole ranking",
			observer.performed,
		)
	}
}

type spreadAnswering struct {
	answers peeranswers.AnsweredQuery
}

func (s spreadAnswering) SpreadOverPeers(
	_ context.Context,
	_ searchquery.Query,
	_ []peerdirectory.AskablePeer,
) peeranswers.AnsweredQuery {
	return s.answers
}

func answersOfTwoWords(t *testing.T, commonWordAddress, rareWordAddress string) spreadAnswering {
	t.Helper()

	return spreadAnswering{answers: peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachAnswer: [][]peeranswers.AnsweredItem{
			{answeredItemCountedForTheWord(t, commonWordAddress, "berlin")},
			{answeredItemCountedForTheWord(t, rareWordAddress, "kelondro")},
		},
		DocumentsHeldPerQueryWord: map[yacymodel.Hash]int{
			yacymodel.WordHash("berlin"):   100000,
			yacymodel.WordHash("kelondro"): 10,
		},
	}}
}

func answeredItemCountedForTheWord(
	t *testing.T, address string, word string,
) peeranswers.AnsweredItem {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return peeranswers.AnsweredItem{
		Metadata: yacymodel.URLMetadata{Hash: hash, Address: address},
		MatchedWords: map[yacymodel.Hash]peeranswers.WordCount{
			yacymodel.WordHash(word): {Hits: 1},
		},
	}
}

func TestTheRankingByRelevancePutsTheRarerWordFirst(t *testing.T) {
	t.Parallel()

	common, rare := "https://common.example/", "https://rare.example/"
	network := networkOrdering(
		t,
		directoryAnsweringAt(t, peerHolding(t)),
		&recordedQuery{},
		answersOfTwoWords(t, common, rare),
		relevance.New(relevance.DefaultScoreWeights()),
	)

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro"))

	if len(ranking.Items) != 2 || ranking.Items[0].Address != rare {
		t.Fatalf("the ranking reads %+v, want the document of the rarer word first", ranking.Items)
	}
}

func TestASearchReportsHowManyRankedItemsAPeerCounted(t *testing.T) {
	t.Parallel()

	common, rare := "https://common.example/", "https://rare.example/"
	observer := &recordedQuery{}
	network := networkSearching(
		t, directoryAnsweringAt(t, peerHolding(t)), observer, answersOfTwoWords(t, common, rare),
	)

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro"))

	if len(ranking.Items) != 2 || ranking.Items[0].Address != common {
		t.Fatalf("the ranking reads %+v, want the order the peers put", ranking.Items)
	}
	if observer.performed.AmountOfRankedItemsCountedByAPeer != 2 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want both ranked items counted by a peer",
			observer.performed,
		)
	}
}

func answersOfTwoWordsMatchedByEveryItem(
	t *testing.T, commonWordAddress, rareWordAddress string,
) spreadAnswering {
	t.Helper()

	queryWords := []yacymodel.Hash{
		yacymodel.WordHash("berlin"), yacymodel.WordHash("kelondro"),
	}
	answers := answersOfTwoWords(t, commonWordAddress, rareWordAddress)
	for _, itemsOfOneAnswer := range answers.answers.ItemsInTheOrderOfEachAnswer {
		for place, item := range itemsOfOneAnswer {
			itemsOfOneAnswer[place] = item.MatchingTheWords(queryWords)
		}
	}

	return answers
}

type pagesHoldingTheWordOfOneDocument struct {
	address string
	word    string
	hits    int
}

func (p pagesHoldingTheWordOfOneDocument) DocumentTextPerDocument(
	_ context.Context,
	_ []yacymodel.Hash,
	pagesToRead []pagereading.PageToRead,
) map[yacymodel.URLHash]documenttext.DocumentText {
	documentTextPerDocument := map[yacymodel.URLHash]documenttext.DocumentText{}
	for _, pageToRead := range pagesToRead {
		if pageToRead.Address != p.address {
			continue
		}
		documentTextPerDocument[pageToRead.Document] = documenttext.DocumentText{
			HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash(p.word): p.hits},
			AmountOfWords:    p.hits,
		}
	}

	return documentTextPerDocument
}

func TestTheRankingByRelevanceFollowsTheWordsReadFromThePages(t *testing.T) {
	t.Parallel()

	common, rare := "https://common.example/", "https://rare.example/"
	network := networksearch.New(
		directoryAnsweringAt(t, peerHolding(t)),
		answersOfTwoWordsMatchedByEveryItem(t, common, rare),
		pagesHoldingTheWordOfOneDocument{address: common, word: "kelondro", hits: 50},
		relevance.New(relevance.DefaultScoreWeights()),
		queryBudget,
		pageReadBudget,
		pagesReadPerQuery,
		recordCeiling,
		networksearch.NetworkSearchObservers{&recordedQuery{}},
	)

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro"))

	if len(ranking.Items) != 2 || ranking.Items[0].Address != common {
		t.Fatalf(
			"the ranking reads %+v, want the document whose page holds the rarer word first",
			ranking.Items,
		)
	}
}

type recordedBudgets struct {
	spread      time.Duration
	pageReading time.Duration
}

type spreadRecordingTheBudgetItGets struct {
	answers  peeranswers.AnsweredQuery
	recorded *recordedBudgets
}

func (s spreadRecordingTheBudgetItGets) SpreadOverPeers(
	ctx context.Context,
	_ searchquery.Query,
	_ []peerdirectory.AskablePeer,
) peeranswers.AnsweredQuery {
	s.recorded.spread = budgetLeftIn(ctx)

	return s.answers
}

type pagesRecordingTheBudgetTheyGet struct {
	recorded *recordedBudgets
}

func (p pagesRecordingTheBudgetTheyGet) DocumentTextPerDocument(
	ctx context.Context,
	_ []yacymodel.Hash,
	_ []pagereading.PageToRead,
) map[yacymodel.URLHash]documenttext.DocumentText {
	p.recorded.pageReading = budgetLeftIn(ctx)

	return nil
}

func budgetLeftIn(ctx context.Context) time.Duration {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return 0
	}

	return time.Until(deadline)
}

func networkRecordingItsBudgets(
	t *testing.T,
	recorded *recordedBudgets,
	pageReadBudgetOfTheQuery time.Duration,
) networksearch.Network {
	t.Helper()

	return networksearch.New(
		directoryAnsweringAt(t, peerHolding(t)),
		spreadRecordingTheBudgetItGets{
			answers:  answersOfTwoWords(t, "https://a.example/", "https://b.example/").answers,
			recorded: recorded,
		},
		pagesRecordingTheBudgetTheyGet{recorded: recorded},
		peerorder.Ordering{},
		queryBudget,
		pageReadBudgetOfTheQuery,
		pagesReadPerQuery,
		recordCeiling,
		networksearch.NetworkSearchObservers{&recordedQuery{}},
	)
}

const budgetReadingTolerance = 500 * time.Millisecond

func TestTheQuerySpreadLeavesThePageReadBudgetToThePages(t *testing.T) {
	t.Parallel()

	recorded := &recordedBudgets{}
	network := networkRecordingItsBudgets(t, recorded, pageReadBudget)

	network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro"))

	spreadBudget := queryBudget - pageReadBudget
	if recorded.spread > spreadBudget ||
		recorded.spread < spreadBudget-budgetReadingTolerance {
		t.Fatalf(
			"the spread got %v, want the query budget less the page read budget of %v",
			recorded.spread, spreadBudget,
		)
	}
	if recorded.pageReading < pageReadBudget {
		t.Fatalf(
			"the pages got %v, want the page read budget of %v",
			recorded.pageReading, pageReadBudget,
		)
	}
}

func TestAPageReadBudgetOfTheWholeQueryLeavesTheQuerySpreadNothing(t *testing.T) {
	t.Parallel()

	recorded := &recordedBudgets{}
	network := networkRecordingItsBudgets(t, recorded, queryBudget)

	network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro"))

	if recorded.spread > 0 {
		t.Fatalf("the spread got %v, want nothing left for it", recorded.spread)
	}
	if recorded.pageReading < queryBudget-budgetReadingTolerance {
		t.Fatalf(
			"the pages got %v, want what is left of the query budget of %v",
			recorded.pageReading, queryBudget,
		)
	}
}
