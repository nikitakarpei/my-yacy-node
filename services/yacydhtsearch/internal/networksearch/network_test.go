package networksearch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearchwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	networkName    = "freeworld"
	responseLimit  = 1 << 20
	callsInFlight  = 4
	queryBudget    = 5 * time.Second
	peerCallBudget = 3 * time.Second
	peerResults    = 10
	directoryLimit = 16
	recordCeiling  = 50
	cooldown       = 5 * time.Second
)

type silentDirectoryObserver struct{}

func (silentDirectoryObserver) PeerAdmitted(context.Context, yacymodel.Hash, int)     {}
func (silentDirectoryObserver) PeerAnswering(context.Context, yacymodel.Hash, string) {}
func (silentDirectoryObserver) PeerSilent(context.Context, yacymodel.Hash)            {}
func (silentDirectoryObserver) PeerDropped(context.Context, yacymodel.Hash)           {}
func (silentDirectoryObserver) DirectoryHolds(context.Context, int, int, int)         {}

type silentOutcome struct{}

func (silentOutcome) PeerAnswered(
	context.Context, string, []searchresult.Item, time.Duration,
) {
}
func (silentOutcome) PeerRefused(context.Context, string, int, time.Duration)            {}
func (silentOutcome) PeerUnreachable(context.Context, string, error, time.Duration)      {}
func (silentOutcome) PeerAnswerUnreadable(context.Context, string, error, time.Duration) {}

type recordedQuery struct {
	performed          networksearch.PerformedNetworkSearch
	withoutPeers       int
	withoutIndexedTerm int
}

func (r *recordedQuery) NetworkSearchPerformed(
	_ context.Context,
	search networksearch.PerformedNetworkSearch,
) {
	r.performed = search
}

func (r *recordedQuery) NetworkSearchFoundNoAskablePeers(context.Context) { r.withoutPeers++ }

func (r *recordedQuery) NetworkSearchFoundNoIndexedTerm(context.Context) {
	r.withoutIndexedTerm++
}

type everyAskablePeer struct{}

func (everyAskablePeer) PeersFor(
	_ context.Context,
	_ searchquery.Query,
	askable []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	return askable
}

type noPeerAtAll struct{}

func (noPeerAtAll) PeersFor(
	context.Context,
	searchquery.Query,
	[]peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	return nil
}

func peerHolding(t *testing.T, addresses ...string) string {
	t.Helper()

	resources := make([]yacyproto.SearchResource, 0, len(addresses))
	for _, address := range addresses {
		resources = append(resources, yacyproto.SearchResource{
			Metadata: yacymodel.URLMetadata{Address: address},
		})
	}
	body := yacyproto.SearchResponse{Count: len(resources), Resources: resources}.Encode().Encode()

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
	selection networksearch.PeerSelection,
	observer networksearch.NetworkSearchObserver,
) networksearch.Network {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("partitions from exponent: %v", err)
	}

	return networksearch.New(
		networkName,
		directory,
		selection,
		peersearch.New(
			peersearchwire.New(http.DefaultClient, responseLimit, silentOutcome{}),
			callsInFlight,
			peerCallBudget,
		),
		queryBudget,
		peerCallBudget,
		peerResults,
		recordCeiling,
		partitions,
		networksearch.NetworkSearchObservers{observer},
	)
}

func TestOneQueryCarriesBackWhatThePeersHold(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t, peerHolding(t, "https://a.example/"))
	network := networkOver(t, directory, everyAskablePeer{}, observer)

	ranking := network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if len(ranking.Items) != 1 || ranking.Items[0].Address != "https://a.example/" {
		t.Fatalf("Search = %+v, want the address the peer holds", ranking.Items)
	}
	if observer.performed.AmountOfAskedPeers != 1 ||
		observer.performed.AmountOfAnsweringPeers != 1 ||
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
	network := networkOver(t, directory, everyAskablePeer{}, &recordedQuery{})

	ranking := network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if len(ranking.Items) != recordCeiling {
		t.Fatalf("Search carried %d items, want the ceiling %d", len(ranking.Items), recordCeiling)
	}
}

func TestAQueryThatReachesNoPeerIsReportedAsSuch(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	network := networkOver(t, directoryAnsweringAt(t), noPeerAtAll{}, observer)

	ranking := network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if len(ranking.Items) != 0 || observer.withoutPeers != 1 {
		t.Fatalf(
			"Search = %+v with %d reports, want an empty ranking and one report",
			ranking.Items,
			observer.withoutPeers,
		)
	}
}

func TestAQueryWithoutAnIndexedTermReachesNoPeer(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t, peerHolding(t, "https://a.example/"))
	network := networkOver(t, directory, everyAskablePeer{}, observer)

	ranking := network.Search(t.Context(), searchquery.QueryFrom("1"))

	if len(ranking.Items) != 0 || observer.performed.AmountOfAskedPeers != 0 {
		t.Fatalf(
			"Search = %+v after asking %d peers, want an empty ranking and no peer asked",
			ranking.Items,
			observer.performed.AmountOfAskedPeers,
		)
	}
	if observer.withoutIndexedTerm != 1 {
		t.Fatalf(
			"NetworkSearchFoundNoIndexedTerm reported %d times, want one",
			observer.withoutIndexedTerm,
		)
	}
}

func TestAnAddressTwoPeersHoldIsCountedOnceAndReportedAsRepeated(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t,
		peerHolding(t, "https://a.example/", "https://shared.example/"),
		peerHolding(t, "https://shared.example/", "https://b.example/"),
	)
	network := networkOver(t, directory, everyAskablePeer{}, observer)

	ranking := network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if len(ranking.Items) != 3 {
		t.Fatalf("Search = %+v, want the three addresses the two peers hold", ranking.Items)
	}
	if observer.performed.AmountOfItemsAcrossAnswers != 4 ||
		observer.performed.AmountOfRepeatedItemsAcrossAnswers != 1 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want four items answered and one repeated",
			observer.performed,
		)
	}
}

func TestPeersThatHoldNoAddressInCommonReportNoRepeatedItem(t *testing.T) {
	t.Parallel()

	observer := &recordedQuery{}
	directory := directoryAnsweringAt(t,
		peerHolding(t, "https://a.example/"),
		peerHolding(t, "https://b.example/"),
	)
	network := networkOver(t, directory, everyAskablePeer{}, observer)

	network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if observer.performed.AmountOfItemsAcrossAnswers != 2 ||
		observer.performed.AmountOfRepeatedItemsAcrossAnswers != 0 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want two items answered and none repeated",
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
	network := networkOver(t, directory, everyAskablePeer{}, observer)

	network.Search(t.Context(), searchquery.QueryFrom("berlin"))

	if observer.performed.AmountOfAnsweringPeers != 2 ||
		observer.performed.AmountOfPeersThatSentItems != 1 {
		t.Fatalf(
			"NetworkSearchPerformed = %+v, want two peers answering and one sending items",
			observer.performed,
		)
	}
}
