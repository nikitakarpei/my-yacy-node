package networksearch_test

import (
	"context"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/relevance"
	hedgedelaysconstant "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/hedgedelays/constant"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	networkName           = "freeworld"
	responseLimit         = 1 << 20
	peerCallsInFlight     = 48
	urlMetadataCallBudget = 3 * time.Second
	searchCallBudget      = 3 * time.Second
	queryBudget           = 5 * time.Second
	pageReadBudget        = 3 * time.Second
	peerResults           = 10
	directoryLimit        = 64
	recordCeiling         = 50
	compoundWordsCeiling  = 4
	pagesReadPerQuery     = 50
	pagesReadPerSite      = pagesReadPerQuery

	networkRedundancy          = 2
	replicasCoveringAPartition = networkRedundancy
	hedgeDelay                 = 50 * time.Millisecond
	documentsToMatchCeiling    = 64
)

type silentDirectoryObserver struct{}

func (silentDirectoryObserver) PeerAdmitted(context.Context, yacymodel.Hash, int)               {}
func (silentDirectoryObserver) PeerAnswered(context.Context, yacymodel.Hash, string, time.Time) {}
func (silentDirectoryObserver) PeerWentSilent(context.Context, yacymodel.Hash)                  {}
func (silentDirectoryObserver) PeerDropped(context.Context, yacymodel.Hash)                     {}
func (silentDirectoryObserver) PeersKnown(context.Context, int, int, int)                       {}

type silentOutcome struct{}

func (silentOutcome) PeerCallWaitsForASlot(context.Context, string, peerasks.AskedFor) {}

func (silentOutcome) PeerCallTookASlot(
	context.Context, string, peerasks.AskedFor, time.Duration,
) {
}

func (silentOutcome) PeerAnsweredURLMetadata(context.Context, string, int, int, time.Duration) {}
func (silentOutcome) PeerSearchedDocuments(
	context.Context, string, int, int, time.Duration,
) {
}

func (silentOutcome) PeerRefused(
	context.Context, string, peerasks.AskedFor, int, int, time.Duration,
) {
}

func (silentOutcome) PeerUnreachable(
	context.Context, string, peerasks.AskedFor, int, error, time.Duration,
) {
}

func (silentOutcome) PeerAnswerUnreadable(
	context.Context, string, peerasks.AskedFor, int, error, time.Duration,
) {
}

func (silentOutcome) PeerCallCancelled(
	context.Context, string, peerasks.AskedFor, int, time.Duration,
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

func (everyAskablePeer) ChosenPeersPerQueryWordFor(
	_ context.Context,
	queryWords []yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
) peerchoice.ChosenPeersPerQueryWord {
	peersPerQueryWord := make(peerchoice.ChosenPeersPerQueryWord, 0, len(queryWords))
	for _, queryWord := range queryWords {
		peersPerQueryWord = append(peersPerQueryWord, peerchoice.ChosenPeersOfQueryWord{
			QueryWord:   queryWord,
			ChosenPeers: peersOfOnePartition(askablePeers),
		})
	}

	return peersPerQueryWord
}

func peersOfOnePartition(
	askablePeers []peerdirectory.AskablePeer,
) []peerchoice.ChosenPeer {
	chosenPeers := make([]peerchoice.ChosenPeer, 0, len(askablePeers))
	for _, peer := range askablePeers {
		chosenPeers = append(chosenPeers, peerchoice.ChosenPeer{Peer: peer, Partition: 0})
	}

	return chosenPeers
}

func peerHolding(t *testing.T, addresses ...string) string {
	t.Helper()

	metadataOfEachDocument := make([]yacymodel.URLMetadata, 0, len(addresses))
	for _, address := range addresses {
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			t.Fatalf("URLHashOf(%q): %v", address, err)
		}
		metadataOfEachDocument = append(
			metadataOfEachDocument,
			yacymodel.URLMetadata{Hash: hash, Address: address, Title: "Weather"},
		)
	}

	return peerHoldingTheMetadata(t, metadataOfEachDocument...)
}

func peerHoldingTheMetadata(t *testing.T, metadataOfEachDocument ...yacymodel.URLMetadata) string {
	t.Helper()

	resources := make([]yacyproto.SearchResource, 0, len(metadataOfEachDocument))
	for _, metadata := range metadataOfEachDocument {
		resources = append(resources, yacyproto.SearchResource{Metadata: metadata})
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
		peerdirectory.DirectoryLimits{Capacity: directoryLimit},
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

func (stalestFirst) StalestPeersFirst(
	_ context.Context,
	members []peerdirectory.KnownPeer,
	candidates []peerdirectory.CandidatePeer,
) []yacymodel.Hash {
	stalest := make([]yacymodel.Hash, 0, len(members)+len(candidates))
	for _, candidate := range candidates {
		stalest = append(stalest, candidate.Hash)
	}
	for _, member := range members {
		stalest = append(stalest, member.Hash)
	}

	return stalest
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
		replicaAsks(t),
		peerResults,
		peermatched.PeerMatchedSpreadObservers{},
	)
}

func wordJoinedSpread(t *testing.T) wordjoined.Spread {
	t.Helper()

	return wordjoined.New(
		replicaAsks(t),
		peerCalls(t),
		noRememberedQueryWordDocumentAmounts{},
		wordjoined.URLMetadataLookupCutoff{},
		rand.UintN,
		recordCeiling,
		documentsToMatchCeiling,
		peerResults,
		ringPartitions(t),
		yacymodel.PeersHoldingOneWordOf(ringPartitions(t), networkRedundancy),
		wordjoined.WordJoinedSpreadObservers{},
	)
}

func replicaAsks(t *testing.T) replicaasks.Asks {
	t.Helper()

	return replicaasks.New(
		peerCalls(t),
		hedgedelaysconstant.New(hedgeDelay),
		replicasCoveringAPartition,
		replicaasks.ReplicaAsksObservers{},
	)
}

func peerCalls(t *testing.T) peercallwire.Wire {
	t.Helper()

	return peercallwire.New(
		http.DefaultClient,
		peercallwire.SearchedNetwork{Name: networkName, RingPartitions: ringPartitions(t)},
		peercallwire.PeerCallLimits{
			MaxResponseBytes:      responseLimit,
			PeerCallsInFlight:     peerCallsInFlight,
			URLMetadataCallBudget: urlMetadataCallBudget,
			SearchCallBudget:      searchCallBudget,
		},
		silentOutcome{},
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

	return networkOrdering(t, directory, observer, querySpread, orderingInTheFoundOrder{})
}

type orderingInTheFoundOrder struct{}

func (orderingInTheFoundOrder) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	return answers.FoundDocuments
}

func networkOrdering(
	t *testing.T,
	directory *peerdirectory.Directory,
	observer networksearch.NetworkSearchObserver,
	querySpread networksearch.QuerySpread,
	documentsOrdering networksearch.DocumentsOrdering,
) networksearch.Network {
	t.Helper()

	return networksearch.New(
		directory,
		everyAskablePeer{},
		querySpread,
		pagesThatNoOneReads{},
		documentsOrdering,
		queryBudget,
		pageReadBudget,
		pagesReadPerQuery,
		pagesReadPerSite,
		recordCeiling,
		compoundWordsCeiling,
		networksearch.NetworkSearchObservers{observer},
	)
}

type pagesThatNoOneReads struct{}

func (pagesThatNoOneReads) ReadEachPage(
	_ context.Context,
	_ []yacymodel.Hash,
	_ []pagereading.PageToRead,
) pagereading.ReadPages {
	return pagereading.ReadPages{}
}

func TestOneQueryCarriesBackWhatThePeersHold(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t, peerHolding(t, "https://a.example/"))
	network := networkOver(t, directory, observer)

	ranking, outcome := network.Search(t.Context(), searchquery.QueryFrom("berlin", ""))

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

func TestARankedItemCarriesWhatThePeerReportedOfItsDocument(t *testing.T) {
	t.Parallel()

	hash, err := yacymodel.URLHashOf("https://a.example/weather")
	if err != nil {
		t.Fatalf("URLHashOf: %v", err)
	}
	modified := yacymodel.NewCalendarDay(2026, time.March, 4)
	directory := directoryAnsweringAt(t, peerHoldingTheMetadata(t, yacymodel.URLMetadata{
		Hash:           hash,
		Address:        "https://a.example/weather",
		Title:          "Weather",
		Snippet:        "Rain in Berlin.",
		Modified:       yacymodel.Some(modified),
		FaviconAddress: "https://a.example/icon.png",
	}))
	network := networkOver(t, directory, &recordedQuery{})

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin", ""))

	if len(ranking.Items) != 1 {
		t.Fatalf("Search = %+v, want the one document the peer holds", ranking.Items)
	}
	item := ranking.Items[0]
	published, _ := item.PublishedAt.Get()
	if item.Hash != hash || item.Title != "Weather" || item.Description != "Rain in Berlin." ||
		item.ImageAddress != "https://a.example/icon.png" || !published.Equal(modified.Time()) {
		t.Fatalf("the ranked item reads %+v, want what the peer reported", item)
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

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin", ""))

	if len(ranking.Items) != recordCeiling {
		t.Fatalf("Search carried %d items, want the ceiling %d", len(ranking.Items), recordCeiling)
	}
}

func TestAQueryThatReachesNoPeerCarriesBackThatOutcome(t *testing.T) {
	t.Parallel()

	network := networkOver(t, directoryAnsweringAt(t), &recordedQuery{})

	ranking, outcome := network.Search(t.Context(), searchquery.QueryFrom("berlin", ""))

	if len(ranking.Items) != 0 || outcome != networksearch.NoPeerToAsk {
		t.Fatalf(
			"Search = %+v with outcome %v, want an empty ranking and no peer to ask",
			ranking.Items,
			outcome,
		)
	}
}

func TestAQueryWithoutAnIndexedWordReachesNoPeer(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t, peerHolding(t, "https://a.example/"))
	network := networkOver(t, directory, observer)

	ranking, outcome := network.Search(t.Context(), searchquery.QueryFrom("1", ""))

	if len(ranking.Items) != 0 || observer.performed.AmountOfAskablePeers != 0 {
		t.Fatalf(
			"Search = %+v after asking %d peers, want an empty ranking and no peer asked",
			ranking.Items,
			observer.performed.AmountOfAskablePeers,
		)
	}
	if outcome != networksearch.NoIndexedWordInQuery {
		t.Fatalf("Search reached outcome %v, want no indexed word in the query", outcome)
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

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin", ""))

	if len(ranking.Items) != 3 {
		t.Fatalf("Search = %+v, want the three addresses the two peers hold", ranking.Items)
	}
	if observer.performed.AmountOfFoundDocuments != 3 ||
		observer.performed.AmountOfItemsInRanking != 3 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want three documents found and three ranked",
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

	network.Search(t.Context(), searchquery.QueryFrom("berlin", ""))

	if observer.performed.AmountOfAskablePeers != 2 ||
		observer.performed.AmountOfFoundDocuments != 1 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want two chosen peers and one found document",
			observer.performed,
		)
	}
}

func TestADocumentOnePeerListsTwiceIsFoundOnce(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t, peerHolding(t, "https://a.example/", "https://a.example/"))
	network := networkSearching(t, directory, observer, peerMatchedSpread(t))

	network.Search(t.Context(), searchquery.QueryFrom("berlin", ""))

	if observer.performed.AmountOfAskablePeers != 1 ||
		observer.performed.AmountOfItemsInRanking != 1 ||
		observer.performed.AmountOfFoundDocuments != 1 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want the document the peer listed twice found once",
			observer.performed,
		)
	}
}

type spreadAnswering struct {
	answers queryanswers.AnsweredQuery
}

func (s spreadAnswering) SpreadOverPeers(
	_ context.Context,
	_ searchquery.Query,
	_ peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	return s.answers
}

func answersOfTwoWords(t *testing.T, commonWordAddress, rareWordAddress string) spreadAnswering {
	t.Helper()

	commonWordDocument := documentOf(t, commonWordAddress)
	rareWordDocument := documentOf(t, rareWordAddress)

	return spreadAnswering{answers: queryanswers.AnsweredQuery{
		QueryWords: []yacymodel.Hash{
			yacymodel.WordHash("berlin"), yacymodel.WordHash("kelondro"),
		},
		FoundDocuments: []queryanswers.FoundDocument{
			{
				Hash:    commonWordDocument,
				Address: commonWordAddress,
				Facts:   oneHitOfTheWord("berlin"),
			},
			{
				Hash:    rareWordDocument,
				Address: rareWordAddress,
				Facts:   oneHitOfTheWord("kelondro"),
			},
		},
		DocumentsHeldPerQueryWord: map[yacymodel.Hash]int{
			yacymodel.WordHash("berlin"):   100000,
			yacymodel.WordHash("kelondro"): 10,
		},
	}}
}

func documentOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return hash
}

func oneHitOfTheWord(word string) queryanswers.DocumentFacts {
	return queryanswers.DocumentFacts{
		HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash(word): 1},
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
		relevance.New(
			documentrelevance.RelevanceScorerWeighedBy(
				documentrelevance.DefaultRelevanceWeights(),
			),
		),
	)

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro", ""))

	if len(ranking.Items) != 2 || ranking.Items[0].Address != rare {
		t.Fatalf("the ranking reads %+v, want the document of the rarer word first", ranking.Items)
	}
}

type pagesHoldingTheWordOfOneDocument struct {
	address string
	word    string
	hits    int
}

func (p pagesHoldingTheWordOfOneDocument) ReadEachPage(
	_ context.Context,
	_ []yacymodel.Hash,
	pagesToRead []pagereading.PageToRead,
) pagereading.ReadPages {
	pageContentsPerDocument := map[yacymodel.URLHash]pagecontents.PageContents{}
	for _, pageToRead := range pagesToRead {
		if pageToRead.Address != p.address {
			continue
		}
		pageContentsPerDocument[pageToRead.Document] = pagecontents.PageContents{
			HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash(p.word): p.hits},
			AmountOfWords:    p.hits,
		}
	}

	return pagereading.ReadPages{PageContentsPerDocument: pageContentsPerDocument}
}

func TestTheRankingByRelevanceFollowsTheWordsReadFromThePages(t *testing.T) {
	t.Parallel()

	common, rare := "https://common.example/", "https://rare.example/"
	network := networksearch.New(
		directoryAnsweringAt(t, peerHolding(t)),
		everyAskablePeer{},
		answersOfTwoWords(t, common, rare),
		pagesHoldingTheWordOfOneDocument{address: common, word: "kelondro", hits: 50},
		relevance.New(
			documentrelevance.RelevanceScorerWeighedBy(
				documentrelevance.DefaultRelevanceWeights(),
			),
		),
		queryBudget,
		pageReadBudget,
		pagesReadPerQuery,
		pagesReadPerSite,
		recordCeiling,
		compoundWordsCeiling,
		networksearch.NetworkSearchObservers{&recordedQuery{}},
	)

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro", ""))

	if len(ranking.Items) != 2 || ranking.Items[0].Address != common {
		t.Fatalf(
			"the ranking reads %+v, want the document whose page holds the rarer word first",
			ranking.Items,
		)
	}
}

type pagesRecordingTheirAddresses struct {
	addresses *[]string
}

func (p pagesRecordingTheirAddresses) ReadEachPage(
	_ context.Context,
	_ []yacymodel.Hash,
	pagesToRead []pagereading.PageToRead,
) pagereading.ReadPages {
	for _, pageToRead := range pagesToRead {
		*p.addresses = append(*p.addresses, pageToRead.Address)
	}

	return pagereading.ReadPages{}
}

func TestNoMorePagesOfOneSiteAreReadThanItsShare(t *testing.T) {
	t.Parallel()

	addresses := []string{
		"https://spam.example/1",
		"https://spam.example/2",
		"https://spam.example/3",
		"https://other.example/",
	}
	foundDocuments := make([]queryanswers.FoundDocument, 0, len(addresses))
	for _, address := range addresses {
		foundDocuments = append(foundDocuments, queryanswers.FoundDocument{
			Hash:    documentOf(t, address),
			Address: address,
			Facts:   oneHitOfTheWord("berlin"),
		})
	}
	var addressesRead []string
	network := networksearch.New(
		directoryAnsweringAt(t, peerHolding(t)),
		everyAskablePeer{},
		spreadAnswering{answers: queryanswers.AnsweredQuery{
			QueryWords:     []yacymodel.Hash{yacymodel.WordHash("berlin")},
			FoundDocuments: foundDocuments,
		}},
		pagesRecordingTheirAddresses{addresses: &addressesRead},
		orderingInTheFoundOrder{},
		queryBudget,
		pageReadBudget,
		pagesReadPerQuery,
		2,
		recordCeiling,
		compoundWordsCeiling,
		networksearch.NetworkSearchObservers{&recordedQuery{}},
	)

	network.Search(t.Context(), searchquery.QueryFrom("berlin", ""))

	if !slices.Equal(addressesRead, []string{
		"https://spam.example/1", "https://spam.example/2", "https://other.example/",
	}) {
		t.Fatalf(
			"the pages read are %v, want two of the one site and the other site",
			addressesRead,
		)
	}
}

type pagesOfOneDocumentGone struct {
	address string
}

func (p pagesOfOneDocumentGone) ReadEachPage(
	_ context.Context,
	_ []yacymodel.Hash,
	pagesToRead []pagereading.PageToRead,
) pagereading.ReadPages {
	goneDocuments := map[yacymodel.URLHash]struct{}{}
	for _, pageToRead := range pagesToRead {
		if pageToRead.Address == p.address {
			goneDocuments[pageToRead.Document] = struct{}{}
		}
	}

	return pagereading.ReadPages{GoneDocuments: goneDocuments}
}

func TestADocumentWhosePageIsGoneLeavesTheRanking(t *testing.T) {
	t.Parallel()

	common, rare := "https://common.example/", "https://rare.example/"
	network := networksearch.New(
		directoryAnsweringAt(t, peerHolding(t)),
		everyAskablePeer{},
		answersOfTwoWords(t, common, rare),
		pagesOfOneDocumentGone{address: common},
		orderingInTheFoundOrder{},
		queryBudget,
		pageReadBudget,
		pagesReadPerQuery,
		pagesReadPerSite,
		recordCeiling,
		compoundWordsCeiling,
		networksearch.NetworkSearchObservers{&recordedQuery{}},
	)

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro", ""))

	if len(ranking.Items) != 1 || ranking.Items[0].Address != rare {
		t.Fatalf(
			"the ranking reads %+v, want only the document whose page is not gone",
			ranking.Items,
		)
	}
}

type recordedBudgets struct {
	spread      time.Duration
	pageReading time.Duration
}

type spreadRecordingTheBudgetItGets struct {
	answers  queryanswers.AnsweredQuery
	recorded *recordedBudgets
}

func (s spreadRecordingTheBudgetItGets) SpreadOverPeers(
	ctx context.Context,
	_ searchquery.Query,
	_ peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	s.recorded.spread = budgetLeftIn(ctx)

	return s.answers
}

type pagesRecordingTheBudgetTheyGet struct {
	recorded *recordedBudgets
}

func (p pagesRecordingTheBudgetTheyGet) ReadEachPage(
	ctx context.Context,
	_ []yacymodel.Hash,
	_ []pagereading.PageToRead,
) pagereading.ReadPages {
	p.recorded.pageReading = budgetLeftIn(ctx)

	return pagereading.ReadPages{}
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
		everyAskablePeer{},
		spreadRecordingTheBudgetItGets{
			answers:  answersOfTwoWords(t, "https://a.example/", "https://b.example/").answers,
			recorded: recorded,
		},
		pagesRecordingTheBudgetTheyGet{recorded: recorded},
		orderingInTheFoundOrder{},
		queryBudget,
		pageReadBudgetOfTheQuery,
		pagesReadPerQuery,
		pagesReadPerSite,
		recordCeiling,
		compoundWordsCeiling,
		networksearch.NetworkSearchObservers{&recordedQuery{}},
	)
}

const budgetReadingTolerance = 500 * time.Millisecond

func TestTheQuerySpreadLeavesThePageReadBudgetToThePages(t *testing.T) {
	t.Parallel()

	recorded := &recordedBudgets{}
	network := networkRecordingItsBudgets(t, recorded, pageReadBudget)

	network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro", ""))

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

	network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro", ""))

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

func peerListingTheAddressForEachWord(t *testing.T, address string, words ...string) string {
	t.Helper()

	documentHash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}
	documentsPerWord := make(map[yacymodel.Hash][]yacymodel.URLHash, len(words))
	documentsHeldPerWord := make(map[yacymodel.Hash]int, len(words))
	for _, word := range words {
		documentsPerWord[yacymodel.WordHash(word)] = []yacymodel.URLHash{documentHash}
		documentsHeldPerWord[yacymodel.WordHash(word)] = 1
	}
	body := yacyproto.SearchResponse{
		Count: 1,
		Resources: []yacyproto.SearchResource{{
			Metadata: yacymodel.URLMetadata{
				Hash: documentHash, Address: address, Title: "Weather",
			},
		}},
		IndexAbstract: documentsPerWord,
		IndexCount:    documentsHeldPerWord,
	}.Encode().Encode()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(body))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func TestAQueryOfTwoWordsCarriesBackWhatTheReplicasListForBothWords(t *testing.T) {
	t.Parallel()

	const address = "https://a.example/"
	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t, slices.Repeat(
		[]string{peerListingTheAddressForEachWord(t, address, "berlin", "kelondro")},
		directoryLimit,
	)...)
	network := networkSearching(t, directory, observer, wordJoinedSpread(t))

	ranking, outcome := network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro", ""))

	if outcome != networksearch.PeersAsked {
		t.Fatalf("Search reached outcome %v, want peers asked", outcome)
	}
	if len(ranking.Items) != 1 || ranking.Items[0].Address != address {
		t.Fatalf("Search = %+v, want the address the replicas list for both words", ranking.Items)
	}
	if observer.performed.AmountOfItemsInRanking != 1 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want the one item both words joined on",
			observer.performed,
		)
	}
}

func TestAQueryOfTwoWordsCarriesBackWhatAReplicaListsForTheirCompoundWord(t *testing.T) {
	t.Parallel()

	const address = "https://a.example/"
	directory := directoryAnsweringAt(t, slices.Repeat(
		[]string{peerListingTheAddressForEachWord(t, address, "berlinkelondro")},
		directoryLimit,
	)...)
	network := networkSearching(t, directory, &recordedQuery{}, wordJoinedSpread(t))

	ranking, _ := network.Search(t.Context(), searchquery.QueryFrom("berlin kelondro", ""))

	if len(ranking.Items) != 1 || ranking.Items[0].Address != address {
		t.Fatalf(
			"Search = %+v, want the address the replica lists for the compound word",
			ranking.Items,
		)
	}
}

type noRememberedQueryWordDocumentAmounts struct{}

func (noRememberedQueryWordDocumentAmounts) DocumentAmountsOf(
	context.Context,
	[]yacymodel.Hash,
) map[yacymodel.Hash]int {
	return nil
}

func (noRememberedQueryWordDocumentAmounts) Remember(context.Context, map[yacymodel.Hash]int) {}
