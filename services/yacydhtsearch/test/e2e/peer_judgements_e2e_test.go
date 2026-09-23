//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/yacypeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	judgementHolderAlias        = "node-judgement-holder-e2e"
	nodeUnderJudgementAlias     = "node-under-judgement-e2e"
	yacyPeerUnderJudgementAlias = "yacy-under-judgement-e2e"
	nodeListingAThousandAlias   = "node-judgement-thousand-e2e"
	yacyPeerHoldingNothingAlias = "yacy-judgement-empty-e2e"
	judgementSearchAlias        = "yacydhtsearch-judgement"

	judgementQuestion = "abstract holds only the documents to match"

	judgementFirstWordToken  = "yacydhtsearchjudgementfirstword"
	judgementSecondWordToken = "yacydhtsearchjudgementsecondword"
	judgementDocumentTitle   = "judged documents probe"

	amountOfFirstWordDocuments              = 200
	amountOfDocumentsOfBothWordsOnTheHolder = 100
	amountOfSecondWordOnlyDocuments         = 150
	amountOfDocumentsListedByANode          = 1000

	partitionExponent = 1
	firstPartition    = 0
	secondPartition   = 1

	smallestStepOnTheRing = 8

	judgementHedgeDelay       = 2 * time.Second
	judgementRankingLifetime  = time.Second
	judgementStalenessHorizon = time.Nanosecond
	pauseBetweenTheQueries    = 5 * time.Second
	directoryRefreshInterval  = 10 * time.Second
	answeringPeersTimeout     = 5 * time.Minute
	judgementTimeout          = 60 * time.Second
)

func TestANodeThatListsOnlyTheCrossCheckedDocumentsIsAskedAgain(t *testing.T) {
	judgeTheUnaskedReplica(t, peerUnderJudgement{
		name:                         nodeUnderJudgementAlias,
		startThePeersBesideTheHolder: startTheNodeLegPeers,
		seedlistURL:                  seedlistURLOf(yacyPeerHoldingNothingAlias),
		amountOfAnsweringPeers:       4,
		judged:                       "honored",
		standing:                     "honoring",
		amountOfAnsweredCrossChecks:  2,
	})
}

func TestAYaCyPeerThatListsMoreThanTheCrossCheckedDocumentsIsNotAskedAgain(t *testing.T) {
	judgeTheUnaskedReplica(t, peerUnderJudgement{
		name:                         yacyPeerUnderJudgementAlias,
		startThePeersBesideTheHolder: startTheYacyLegPeers,
		seedlistURL:                  seedlistURLOf(yacyPeerUnderJudgementAlias),
		amountOfAnsweringPeers:       3,
		judged:                       "ignored",
		standing:                     "ignoring",
		amountOfAnsweredCrossChecks:  1,
	})
}

type peerUnderJudgement struct {
	name                         string
	startThePeersBesideTheHolder peersBesideTheHolderStart
	seedlistURL                  string
	amountOfAnsweringPeers       int
	judged                       string
	standing                     string
	amountOfAnsweredCrossChecks  int
}

type peersBesideTheHolderStart func(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	documents judgementDocuments,
)

type judgementDocuments struct {
	ofTheFirstWord                          []string
	secondWordOnlyOnTheHolder               []string
	secondWordOnlyOnTheJudgedPeer           []string
	secondWordOnlyOnTheNodeListingAThousand []string
}

func (documents judgementDocuments) ofBothWordsOnTheHolder() []string {
	return documents.ofTheFirstWord[:amountOfDocumentsOfBothWordsOnTheHolder]
}

func (documents judgementDocuments) crossChecked() []string {
	return documents.ofTheFirstWord[amountOfDocumentsOfBothWordsOnTheHolder:]
}

func judgeTheUnaskedReplica(t *testing.T, peer peerUnderJudgement) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)

	documents := documentsOfTheJudgement()
	peer.startThePeersBesideTheHolder(t, ctx, probe, network.Name, documents)
	startTheHolderOfBothWords(t, ctx, probe, network.Name, peer.seedlistURL, documents)

	service := startYacydhtsearch(
		t, ctx, network.Name, judgementSearchAlias, peer.seedlistURL, judgementSettings(),
	)
	waitForEveryPeerToAnswer(t, ctx, probe, service, peer.amountOfAnsweringPeers, peer.name)

	query := judgementFirstWordToken + " " + judgementSecondWordToken
	links := resultLinksFor(t, ctx, probe, service.searchURL, query, amountOfFirstWordDocuments)
	waitForMetricLine(t, ctx, probe, service, judgementsLine(peer.judged, 1), peer.name)
	requireCrossCheckedDocumentsAmongTheResults(t, peer.name, links, documents.crossChecked())

	time.Sleep(pauseBetweenTheQueries)
	resultLinksFor(t, ctx, probe, service.searchURL, query, amountOfFirstWordDocuments)
	waitForMetricLine(t, ctx, probe, service, standingsLine(peer.standing, 1), peer.name)
	requireMetricLine(
		t,
		ctx,
		probe,
		service,
		answeredCrossChecksLine(peer.amountOfAnsweredCrossChecks),
		peer.name,
	)
}

func documentsOfTheJudgement() judgementDocuments {
	return judgementDocuments{
		ofTheFirstWord: addressesUnder("first-word", amountOfFirstWordDocuments),
		secondWordOnlyOnTheHolder: addressesUnder(
			"holder-second-word", amountOfSecondWordOnlyDocuments,
		),
		secondWordOnlyOnTheJudgedPeer: addressesUnder(
			"judged-second-word", amountOfSecondWordOnlyDocuments,
		),
		secondWordOnlyOnTheNodeListingAThousand: addressesUnder(
			"thousand-second-word", amountOfDocumentsListedByANode+amountOfSecondWordOnlyDocuments,
		),
	}
}

func addressesUnder(prefix string, amount int) []string {
	addresses := make([]string, amount)
	for i := range addresses {
		addresses[i] = fmt.Sprintf("http://transfer.example.invalid/%s-%04d.html", prefix, i)
	}

	return addresses
}

func startTheNodeLegPeers(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	documents judgementDocuments,
) {
	t.Helper()

	yacypeer.Start(
		t, ctx, probe, networkName, yacyPeerHoldingNothingAlias,
		yacypeer.RemoteSearchOverrides()...,
	)
	startTheNodeListingAThousand(
		t, ctx, probe, networkName, seedlistURLOf(yacyPeerHoldingNothingAlias), documents,
	)

	nodeHash := peerHashJustAfterTheSecondWordIn(t, firstPartition)
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: networkName,
		Alias:       nodeUnderJudgementAlias,
		Hash:        nodeHash,
		SeedlistURL: seedlistURLOf(yacyPeerHoldingNothingAlias),
	})

	addresses := slices.Concat(documents.crossChecked(), documents.secondWordOnlyOnTheJudgedPeer)
	nodepeer.PushPostings(
		t, ctx, probe, nodeURL, nodeHash,
		yacymodel.WordHash(judgementSecondWordToken),
		urlHashesOf(t, addresses),
	)
	nodepeer.PushURLMetadataRows(t, ctx, probe, nodeURL, nodeHash, urlMetadataRowsOf(t, addresses))
}

func startTheNodeListingAThousand(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	seedlistURL string,
	documents judgementDocuments,
) {
	t.Helper()

	nodeHash := peerHashAtTheSecondWordIn(t, firstPartition)
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: networkName,
		Alias:       nodeListingAThousandAlias,
		Hash:        nodeHash,
		SeedlistURL: seedlistURL,
	})
	nodepeer.PushPostings(
		t, ctx, probe, nodeURL, nodeHash,
		yacymodel.WordHash(judgementSecondWordToken),
		urlHashesOf(t, documents.secondWordOnlyOnTheNodeListingAThousand),
	)
	nodepeer.PushURLMetadataRows(
		t, ctx, probe, nodeURL, nodeHash,
		urlMetadataRowsOf(t, documents.secondWordOnlyOnTheNodeListingAThousand),
	)
}

func peerHashAtTheSecondWordIn(t *testing.T, partition uint) yacymodel.Hash {
	t.Helper()

	return yacymodel.HashFromDHTRingPosition(positionOfTheSecondWordIn(t, partition))
}

func positionOfTheSecondWordIn(t *testing.T, partition uint) yacymodel.DHTRingPosition {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("partitions of the exponent %d: %v", partitionExponent, err)
	}

	return yacymodel.DHTRingPositionOfWordInPartition(
		yacymodel.WordHash(judgementSecondWordToken), partition, partitions,
	)
}

func urlHashesOf(t *testing.T, addresses []string) []yacymodel.URLHash {
	t.Helper()

	urlHashes := make([]yacymodel.URLHash, 0, len(addresses))
	for _, address := range addresses {
		urlHashes = append(urlHashes, urlHashOf(t, address))
	}

	return urlHashes
}

func urlHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	urlHash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("url hash of %s: %v", address, err)
	}

	return urlHash
}

func urlMetadataRowsOf(t *testing.T, addresses []string) []yacymodel.URLMetadata {
	t.Helper()

	rows := make([]yacymodel.URLMetadata, 0, len(addresses))
	for _, address := range addresses {
		rows = append(rows, yacymodel.URLMetadata{
			Hash:         urlHashOf(t, address),
			Address:      address,
			Title:        judgementDocumentTitle,
			DocumentType: yacymodel.DocumentTypeHTML,
		})
	}

	return rows
}

func peerHashJustAfterTheSecondWordIn(t *testing.T, partition uint) yacymodel.Hash {
	t.Helper()

	return yacymodel.HashFromDHTRingPosition(
		positionOfTheSecondWordIn(t, partition) + smallestStepOnTheRing,
	)
}

func startTheYacyLegPeers(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	documents judgementDocuments,
) {
	t.Helper()

	_, yacyURL := yacypeer.Start(
		t, ctx, probe, networkName, yacyPeerUnderJudgementAlias,
		yacypeer.RemoteSearchOverrides()...,
	)
	yacypeer.PushDocumentsUnderAddresses(
		t, ctx, probe, yacyURL,
		documents.crossChecked(),
		[]string{judgementSecondWordToken},
	)
	yacypeer.PushDocumentsUnderAddresses(
		t, ctx, probe, yacyURL,
		documents.secondWordOnlyOnTheJudgedPeer,
		[]string{judgementSecondWordToken},
	)
	startTheNodeListingAThousand(
		t, ctx, probe, networkName, seedlistURLOf(yacyPeerUnderJudgementAlias), documents,
	)
}

func startTheHolderOfBothWords(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	seedlistURL string,
	documents judgementDocuments,
) {
	t.Helper()

	nodeHash := peerHashAtTheSecondWordIn(t, secondPartition)
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: networkName,
		Alias:       judgementHolderAlias,
		Hash:        nodeHash,
		SeedlistURL: seedlistURL,
	})

	nodepeer.PushPostings(
		t, ctx, probe, nodeURL, nodeHash,
		yacymodel.WordHash(judgementFirstWordToken),
		urlHashesOf(t, documents.ofTheFirstWord),
	)
	nodepeer.PushPostings(
		t, ctx, probe, nodeURL, nodeHash,
		yacymodel.WordHash(judgementSecondWordToken),
		urlHashesOf(t, slices.Concat(
			documents.ofBothWordsOnTheHolder(), documents.secondWordOnlyOnTheHolder,
		)),
	)
	nodepeer.PushURLMetadataRows(
		t, ctx, probe, nodeURL, nodeHash,
		urlMetadataRowsOf(t, slices.Concat(
			documents.ofTheFirstWord, documents.secondWordOnlyOnTheHolder,
		)),
	)
}

func judgementSettings() map[string]string {
	return map[string]string{
		"YACYDHTSEARCH_PARTITION_EXPONENT":                 strconv.Itoa(partitionExponent),
		"YACYDHTSEARCH_HEDGE_DELAY":                        judgementHedgeDelay.String(),
		"YACYDHTSEARCH_RANKING_LIFETIME":                   judgementRankingLifetime.String(),
		"YACYDHTSEARCH_PEER_RELIABILITY_STALENESS_HORIZON": judgementStalenessHorizon.String(),
		"YACYDHTSEARCH_RANKED_ITEMS_CEILING": strconv.Itoa(
			amountOfFirstWordDocuments,
		),
		"YACYDHTSEARCH_REFRESH_INTERVAL": directoryRefreshInterval.String(),
	}
}

func waitForEveryPeerToAnswer(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	service yacydhtsearchService,
	amountOfAnsweringPeers int,
	peerName string,
) {
	t.Helper()

	everyPeerAnswers := fmt.Sprintf(
		"yacydhtsearch_directory_answering_peers %d", amountOfAnsweringPeers,
	)
	if metricLinePublishedWithin(t, ctx, probe, service, everyPeerAnswers, answeringPeersTimeout) {
		return
	}
	t.Fatalf(
		"%s never held the %d peers beside and including %s as answering peers:\n%s",
		judgementSearchAlias,
		amountOfAnsweringPeers,
		peerName,
		publishedBy(t, ctx, probe, service),
	)
}

func metricLinePublishedWithin(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	service yacydhtsearchService,
	line string,
	within time.Duration,
) bool {
	t.Helper()

	return pollwait.For(within, func() bool {
		return strings.Contains(publishedBy(t, ctx, probe, service), line)
	})
}

func waitForMetricLine(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	service yacydhtsearchService,
	line string,
	peerName string,
) {
	t.Helper()

	if metricLinePublishedWithin(t, ctx, probe, service, line, judgementTimeout) {
		return
	}
	t.Fatalf(
		"%s never published %q for the peer under judgement %s:\n%s",
		judgementSearchAlias,
		line,
		peerName,
		publishedBy(t, ctx, probe, service),
	)
}

func judgementsLine(judged string, counted int) string {
	return fmt.Sprintf(
		`yacydhtsearch_word_joined_spread_peer_judgements_total{judged=%q,question=%q} %d`,
		judged,
		judgementQuestion,
		counted,
	)
}

func requireCrossCheckedDocumentsAmongTheResults(
	t *testing.T,
	peerName string,
	links, crossChecked []string,
) {
	t.Helper()

	if amountOfDocumentsIn(links, crossChecked) > 0 {
		return
	}
	t.Fatalf(
		"%s put none of the %d cross-checked documents among the results; %d results came back",
		peerName,
		len(crossChecked),
		len(links),
	)
}

func amountOfDocumentsIn(links, documents []string) int {
	amount := 0
	for _, document := range documents {
		if slices.Contains(links, document) {
			amount++
		}
	}

	return amount
}

func standingsLine(standing string, counted int) string {
	return fmt.Sprintf(
		`yacydhtsearch_word_joined_spread_peer_standings_total{question=%q,standing=%q} %d`,
		judgementQuestion,
		standing,
		counted,
	)
}

func requireMetricLine(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	service yacydhtsearchService,
	line string,
	peerName string,
) {
	t.Helper()

	metrics := publishedBy(t, ctx, probe, service)
	if strings.Contains(metrics, line) {
		return
	}
	t.Fatalf(
		"%s published no %q for the peer under judgement %s:\n%s",
		judgementSearchAlias,
		line,
		peerName,
		metrics,
	)
}

func answeredCrossChecksLine(counted int) string {
	return fmt.Sprintf(
		`yacydhtsearch_peer_calls_total{asked_for="cross-checked documents",outcome="answered"} %d`,
		counted,
	)
}
