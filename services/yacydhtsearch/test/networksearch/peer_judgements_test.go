package networksearch_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	hedgedelaysconstant "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/hedgedelays/constant"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	peerjudgementledgersmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementledgers/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	judgementFirstWord  = "judgementfirstword"
	judgementSecondWord = "judgementsecondword"

	judgementPartitionExponent   = 1
	judgementNetworkRedundancy   = 2
	judgementReplicasCovering    = 1
	judgementHedgeDelay          = 2 * time.Second
	judgementQueryBudget         = 6 * time.Second
	judgementItemsCeiling        = 10
	judgementDocumentsCeiling    = 1000
	judgementAbstractCeiling     = 1000
	amountOfFirstWordDocuments   = 200
	amountOfSharedDocuments      = 100
	amountOfSecondWordOnlyOnPeer = 150
	amountOfDocumentsOfTheLister = 1150
)

func TestAPeerThatListsOnlyTheCrossCheckedDocumentsIsJudgedHonoringAndAskedAgain(
	t *testing.T,
) {
	t.Parallel()

	network := judgementNetworkOf(t, peerUnderJudgementHonoringTheFilter)

	firstSpread := network.spreadTheQuery(t)
	secondSpread := network.spreadTheQuery(t)

	network.requireJudgedIn(t, firstSpread, peerjudgements.Honored)
	network.requireStandingIn(t, secondSpread, peerjudgements.Honoring)
	if crossChecks := network.peerUnderJudgement.amountOfCrossChecks(); crossChecks != 2 {
		t.Fatalf(
			"the peer under judgement was asked to cross-check %d times, want by both queries",
			crossChecks,
		)
	}
}

func TestAPeerThatListsMoreThanTheCrossCheckedDocumentsIsJudgedIgnoringAndNotAskedAgain(
	t *testing.T,
) {
	t.Parallel()

	network := judgementNetworkOf(t, peerUnderJudgementIgnoringTheFilter)

	firstSpread := network.spreadTheQuery(t)
	secondSpread := network.spreadTheQuery(t)

	network.requireJudgedIn(t, firstSpread, peerjudgements.Ignored)
	network.requireStandingIn(t, secondSpread, peerjudgements.Ignoring)
	if crossChecks := network.peerUnderJudgement.amountOfCrossChecks(); crossChecks != 1 {
		t.Fatalf(
			"the peer under judgement was asked to cross-check %d times, want by the first query only",
			crossChecks,
		)
	}
}

type filterRule bool

const (
	peerUnderJudgementHonoringTheFilter filterRule = true
	peerUnderJudgementIgnoringTheFilter filterRule = false
)

type judgementNetwork struct {
	partitions         yacymodel.DHTRingPartitions
	peers              []peerdirectory.AskablePeer
	peerUnderJudgement *peerHoldingDocuments
	spread             wordjoined.Spread
	spreads            *recordedSpreadsOfTheJudgement
}

func judgementNetworkOf(t *testing.T, filterRule filterRule) judgementNetwork {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(judgementPartitionExponent)
	if err != nil {
		t.Fatalf("partitions from exponent: %v", err)
	}
	secondWord := yacymodel.WordHash(judgementSecondWord)
	peerUnderJudgement := &peerHoldingDocuments{
		hash:            hashOfTheReplica(partitions, secondWord, 0, secondReplica),
		honorsTheFilter: bool(filterRule),
		documentsPerWord: map[yacymodel.Hash][]string{secondWord: slices.Concat(
			firstWordDocuments()[amountOfSharedDocuments:],
			addressesUnder("judged-second-word", amountOfSecondWordOnlyOnPeer),
		)},
	}
	peersHoldingDocuments := append(
		peersBesideThePeerUnderJudgement(partitions), peerUnderJudgement,
	)
	peers := make([]peerdirectory.AskablePeer, 0, len(peersHoldingDocuments))
	for _, peer := range peersHoldingDocuments {
		peers = append(
			peers,
			peerdirectory.AskablePeer{Hash: peer.hash, Address: peer.answeringAt(t)},
		)
	}
	spreads := &recordedSpreadsOfTheJudgement{}

	return judgementNetwork{
		partitions:         partitions,
		peers:              peers,
		peerUnderJudgement: peerUnderJudgement,
		spread:             judgingSpreadOver(partitions, spreads),
		spreads:            spreads,
	}
}

type replicaPlace yacymodel.DHTRingPosition

const (
	firstReplica  replicaPlace = 0
	secondReplica replicaPlace = 8
)

func hashOfTheReplica(
	partitions yacymodel.DHTRingPartitions,
	word yacymodel.Hash,
	partition uint,
	place replicaPlace,
) yacymodel.Hash {
	return yacymodel.HashFromDHTRingPosition(
		yacymodel.DHTRingPositionOfWordInPartition(word, partition, partitions) +
			yacymodel.DHTRingPosition(place),
	)
}

func peersBesideThePeerUnderJudgement(
	partitions yacymodel.DHTRingPartitions,
) []*peerHoldingDocuments {
	firstWord := yacymodel.WordHash(judgementFirstWord)
	secondWord := yacymodel.WordHash(judgementSecondWord)
	compoundWord := judgementQuery().HashesOfWordsAndCompoundWordsUpTo(1)[2]

	return []*peerHoldingDocuments{
		{
			hash: hashOfTheReplica(partitions, firstWord, 0, firstReplica),
			documentsPerWord: map[yacymodel.Hash][]string{
				firstWord: firstWordDocuments(),
				secondWord: slices.Concat(
					firstWordDocuments()[:amountOfSharedDocuments],
					addressesUnder("holder-second-word", amountOfSecondWordOnlyOnPeer),
				),
			},
		},
		{hash: hashOfTheReplica(partitions, firstWord, 0, secondReplica)},
		{
			hash:             hashOfTheReplica(partitions, firstWord, 1, firstReplica),
			documentsPerWord: map[yacymodel.Hash][]string{firstWord: firstWordDocuments()},
		},
		{
			hash: hashOfTheReplica(partitions, secondWord, 0, firstReplica),
			documentsPerWord: map[yacymodel.Hash][]string{
				secondWord: addressesUnder("lister-second-word", amountOfDocumentsOfTheLister),
			},
		},
		{
			hash: hashOfTheReplica(partitions, secondWord, 1, firstReplica),
			documentsPerWord: map[yacymodel.Hash][]string{
				secondWord: addressesUnder(
					"second-partition-second-word",
					amountOfSecondWordOnlyOnPeer,
				),
			},
		},
		{
			hash: hashOfTheReplica(partitions, compoundWord, 0, firstReplica),
			documentsPerWord: map[yacymodel.Hash][]string{
				compoundWord: addressesUnder("compound-word", 1),
			},
		},
	}
}

func firstWordDocuments() []string {
	return addressesUnder("first-word", amountOfFirstWordDocuments)
}

func judgementQuery() searchquery.Query {
	return searchquery.QueryFrom(judgementFirstWord+" "+judgementSecondWord, "")
}

func addressesUnder(prefix string, amount int) []string {
	addresses := make([]string, amount)
	for index := range addresses {
		addresses[index] = fmt.Sprintf("https://judged.example/%s-%04d.html", prefix, index)
	}

	return addresses
}

func judgingSpreadOver(
	partitions yacymodel.DHTRingPartitions,
	spreads *recordedSpreadsOfTheJudgement,
) wordjoined.Spread {
	wire := peercallwire.New(
		http.DefaultClient,
		peercallwire.SearchedNetwork{Name: networkName, RingPartitions: partitions},
		peercallwire.PeerCallLimits{
			MaxResponseBytes:  responseLimit,
			PeerCallsInFlight: peerCallsInFlight,
			PeerCallBudget:    peerCallBudget,
		},
		silentOutcome{},
	)

	return wordjoined.New(
		replicaasks.New(
			wire,
			hedgedelaysconstant.New(judgementHedgeDelay),
			judgementReplicasCovering,
			replicaasks.ReplicaAsksObservers{},
		),
		wire,
		peerjudgements.New(
			wordjoined.ListsOnlyTheCrossCheckedDocuments,
			peerjudgementledgersmemory.New(judgementLedgerCapacity),
			peerRetrialInterval,
			time.Now,
		),
		judgementDocumentsCeiling,
		judgementDocumentsCeiling,
		judgementItemsCeiling,
		partitions,
		yacymodel.PeersHoldingOneWordOf(partitions, judgementNetworkRedundancy),
		wordjoined.WordJoinedSpreadObservers{spreads},
	)
}

func (network judgementNetwork) spreadTheQuery(t *testing.T) wordjoined.PerformedWordJoinedSpread {
	t.Helper()

	query := judgementQuery()
	ctx, endQuery := context.WithTimeout(t.Context(), judgementQueryBudget)
	defer endQuery()
	network.spread.SpreadOverPeers(ctx, query, peerchoice.New(
		network.partitions,
		judgementNetworkRedundancy,
		unknownReliability{},
		silentPeerChoice{},
	).ChosenPeersPerQueryWordFor(ctx, query.HashesOfWordsAndCompoundWordsUpTo(1), network.peers))

	return network.spreads.last(t)
}

func (network judgementNetwork) requireJudgedIn(
	t *testing.T,
	spread wordjoined.PerformedWordJoinedSpread,
	judgement peerjudgements.Judgement,
) {
	t.Helper()

	for _, judgedPeer := range spread.CrossCheckedDocumentsRound.JudgedPeers {
		if judgedPeer.Peer == network.peerUnderJudgement.hash && judgedPeer.Judgement == judgement {
			return
		}
	}
	t.Fatalf(
		"the first query judged %+v, want the peer under judgement %s",
		spread.CrossCheckedDocumentsRound.JudgedPeers,
		judgement,
	)
}

func (network judgementNetwork) requireStandingIn(
	t *testing.T,
	spread wordjoined.PerformedWordJoinedSpread,
	standing peerjudgements.Standing,
) {
	t.Helper()

	for _, peerStanding := range spread.PeerStandings {
		if peerStanding.Peer == network.peerUnderJudgement.hash &&
			peerStanding.Standing == standing {
			return
		}
	}
	t.Fatalf(
		"the second query found the standings %+v, want the peer under judgement %s",
		spread.PeerStandings,
		standing,
	)
}

type peerHoldingDocuments struct {
	hash             yacymodel.Hash
	documentsPerWord map[yacymodel.Hash][]string
	honorsTheFilter  bool
	mutex            sync.Mutex
	crossChecks      int
}

func (peer *peerHoldingDocuments) answeringAt(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if err := request.ParseForm(); err != nil {
				t.Errorf("parse the search form: %v", err)

				return
			}
			searchRequest, err := yacyproto.ParseSearchRequest(request.Context(), request.Form)
			if err != nil {
				t.Errorf("parse the search request: %v", err)

				return
			}
			//nolint:gosec // G705: the answer goes to the test's own peer call, never to a browser
			_, _ = writer.Write([]byte(peer.answerTo(t, searchRequest).Encode().Encode()))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func (peer *peerHoldingDocuments) answerTo(
	t *testing.T,
	searchRequest yacyproto.SearchRequest,
) yacyproto.SearchResponse {
	t.Helper()

	if len(searchRequest.URLs) > 0 {
		peer.mutex.Lock()
		peer.crossChecks++
		peer.mutex.Unlock()
	}
	response := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash][]yacymodel.URLHash{},
		IndexCount:    map[yacymodel.Hash]int{},
	}
	for _, word := range searchRequest.Query {
		documents := peer.documentsPerWord[word]
		if len(documents) == 0 {
			continue
		}
		response.IndexCount[word] = len(documents)
		response.IndexAbstract[word] = peer.documentsListedAmong(t, documents, searchRequest)
		response.Resources = matchedResourcesAmong(t, documents, searchRequest)
	}
	response.Count = len(response.Resources)

	return response
}

func (peer *peerHoldingDocuments) documentsListedAmong(
	t *testing.T,
	documents []string,
	searchRequest yacyproto.SearchRequest,
) []yacymodel.URLHash {
	t.Helper()

	listed := make([]yacymodel.URLHash, 0, len(documents))
	for _, address := range documents {
		document := documentAt(t, address)
		if peer.honorsTheFilter && len(searchRequest.URLs) > 0 &&
			!slices.Contains(searchRequest.URLs, document) {
			continue
		}
		listed = append(listed, document)
		if len(searchRequest.URLs) == 0 && len(listed) == judgementAbstractCeiling {
			break
		}
	}

	return listed
}

func matchedResourcesAmong(
	t *testing.T,
	documents []string,
	searchRequest yacyproto.SearchRequest,
) []yacyproto.SearchResource {
	t.Helper()

	resources := make([]yacyproto.SearchResource, 0, searchRequest.Count)
	for _, address := range documents {
		document := documentAt(t, address)
		if len(searchRequest.URLs) > 0 && !slices.Contains(searchRequest.URLs, document) {
			continue
		}
		if len(resources) == searchRequest.Count {
			break
		}
		resources = append(resources, yacyproto.SearchResource{
			Metadata: yacymodel.URLMetadata{Hash: document, Address: address, Title: "Judged"},
		})
	}

	return resources
}

func documentAt(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	document, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return document
}

func (peer *peerHoldingDocuments) amountOfCrossChecks() int {
	peer.mutex.Lock()
	defer peer.mutex.Unlock()

	return peer.crossChecks
}

type recordedSpreadsOfTheJudgement struct {
	mutex   sync.Mutex
	spreads []wordjoined.PerformedWordJoinedSpread
}

func (recorded *recordedSpreadsOfTheJudgement) WordJoinedSpreadPerformed(
	_ context.Context,
	spread wordjoined.PerformedWordJoinedSpread,
) {
	recorded.mutex.Lock()
	defer recorded.mutex.Unlock()

	recorded.spreads = append(recorded.spreads, spread)
}

func (recorded *recordedSpreadsOfTheJudgement) last(
	t *testing.T,
) wordjoined.PerformedWordJoinedSpread {
	t.Helper()
	recorded.mutex.Lock()
	defer recorded.mutex.Unlock()

	if len(recorded.spreads) == 0 {
		t.Fatal("the spread reported no performed spread")
	}

	return recorded.spreads[len(recorded.spreads)-1]
}

type unknownReliability struct{}

func (unknownReliability) ReliabilityOf(context.Context, probeanswerhistory.PeerAtAddress) float64 {
	return 0
}

type silentPeerChoice struct{}

func (silentPeerChoice) PeersTakenFromTheRing(context.Context, []float64) {}
