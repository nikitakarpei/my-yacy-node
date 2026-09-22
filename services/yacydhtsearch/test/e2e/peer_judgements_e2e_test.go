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

	judgementQuestion = "lists only the cross-checked documents"

	judgementFirstWordToken  = "yacydhtsearchjudgementfirstword"
	judgementSecondWordToken = "yacydhtsearchjudgementsecondword"
	judgementDocumentTitle   = "judged documents probe"

	amountOfNamedDocuments            = 200
	amountOfNamedDocumentsOnTheHolder = 100
	amountOfOtherDocuments            = 150
	amountOfDocumentsListedByANode    = 1000

	partitionExponent = 1
	firstPartition    = 0
	secondPartition   = 1

	smallestStepOnTheRing = 8

	judgementHedgeDelay      = 2 * time.Second
	judgementRankingLifetime = time.Second
	pauseBetweenTheQueries   = 5 * time.Second
	directoryRefreshInterval = 10 * time.Second
	answeringPeersTimeout    = 5 * time.Minute
	judgementTimeout         = 60 * time.Second
)

func TestANodeThatHonorsTheNamedDocumentsIsAskedAgain(t *testing.T) {
	judgeTheUnaskedReplica(t, peerUnderJudgement{
		name:                nodeUnderJudgementAlias,
		startTheOtherPeers:  startTheNodeLegPeers,
		seedlistURL:         seedlistURLOf(yacyPeerHoldingNothingAlias),
		judged:              "honored",
		standing:            "honoring",
		answeredCrossChecks: 2,
	})
}

func TestAYaCyPeerThatIgnoresTheNamedDocumentsIsNotAskedAgain(t *testing.T) {
	judgeTheUnaskedReplica(t, peerUnderJudgement{
		name:                yacyPeerUnderJudgementAlias,
		startTheOtherPeers:  startTheYacyLegPeers,
		seedlistURL:         seedlistURLOf(yacyPeerUnderJudgementAlias),
		judged:              "ignored",
		standing:            "ignoring",
		answeredCrossChecks: 1,
	})
}

type peerUnderJudgement struct {
	name                string
	startTheOtherPeers  otherPeersStart
	seedlistURL         string
	judged              string
	standing            string
	answeredCrossChecks int
}

type otherPeersStart func(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	documents judgementDocuments,
)

type judgementDocuments struct {
	named                          []string
	otherOfTheHolder               []string
	otherOfTheJudgedPeer           []string
	otherOfTheNodeListingAThousand []string
}

func judgeTheUnaskedReplica(t *testing.T, peer peerUnderJudgement) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)

	documents := documentsOfTheJudgement()
	peer.startTheOtherPeers(t, ctx, probe, network.Name, documents)
	startTheHolderOfBothWords(t, ctx, probe, network.Name, peer.seedlistURL, documents)

	service := startYacydhtsearch(
		t, ctx, network.Name, judgementSearchAlias, peer.seedlistURL, judgementSettings(),
	)
	waitForEveryPeerToAnswer(t, ctx, probe, service, peer.name)

	query := judgementFirstWordToken + " " + judgementSecondWordToken
	links := resultLinksFor(t, ctx, probe, service.searchURL, query, amountOfNamedDocuments)
	waitForMetricLine(t, ctx, probe, service, judgementsLine(peer.judged, 1), peer.name)
	requireMoreNamedDocumentsThanTheHolderLists(t, peer.name, links, documents.named)

	time.Sleep(pauseBetweenTheQueries)
	resultLinksFor(t, ctx, probe, service.searchURL, query, amountOfNamedDocuments)
	waitForMetricLine(t, ctx, probe, service, standingsLine(peer.standing, 1), peer.name)
	requireMetricLine(
		t, ctx, probe, service, answeredCrossChecksLine(peer.answeredCrossChecks), peer.name,
	)
}

func documentsOfTheJudgement() judgementDocuments {
	return judgementDocuments{
		named:                addressesUnder("named", amountOfNamedDocuments),
		otherOfTheHolder:     addressesUnder("holder-other", amountOfOtherDocuments),
		otherOfTheJudgedPeer: addressesUnder("judged-other", amountOfOtherDocuments),
		otherOfTheNodeListingAThousand: addressesUnder(
			"thousand-other", amountOfDocumentsListedByANode+amountOfOtherDocuments,
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

	nodeHash := peerHashJustBeforeTheSecondWordIn(t, secondPartition)
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: networkName,
		Alias:       nodeUnderJudgementAlias,
		Hash:        nodeHash,
		SeedlistURL: seedlistURLOf(yacyPeerHoldingNothingAlias),
	})

	addresses := slices.Concat(
		documents.named[amountOfNamedDocumentsOnTheHolder:],
		documents.otherOfTheJudgedPeer,
	)
	nodepeer.PushPostings(
		t, ctx, probe, nodeURL, nodeHash,
		yacymodel.WordHash(judgementSecondWordToken),
		urlHashesOf(t, addresses),
	)
	nodepeer.PushURLMetadataRows(t, ctx, probe, nodeURL, nodeHash, urlMetadataRowsOf(t, addresses))
}

func peerHashJustBeforeTheSecondWordIn(t *testing.T, partition uint) yacymodel.Hash {
	t.Helper()

	return yacymodel.HashFromDHTRingPosition(
		positionOfTheSecondWordIn(t, partition) - smallestStepOnTheRing,
	)
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
		documents.named[amountOfNamedDocumentsOnTheHolder:],
		[]string{judgementSecondWordToken},
	)
	yacypeer.PushDocumentsUnderAddresses(
		t, ctx, probe, yacyURL,
		documents.otherOfTheJudgedPeer,
		[]string{judgementSecondWordToken},
	)

	nodeHash := peerHashAtTheSecondWordIn(t, secondPartition)
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: networkName,
		Alias:       nodeListingAThousandAlias,
		Hash:        nodeHash,
		SeedlistURL: seedlistURLOf(yacyPeerUnderJudgementAlias),
	})
	nodepeer.PushPostings(
		t, ctx, probe, nodeURL, nodeHash,
		yacymodel.WordHash(judgementSecondWordToken),
		urlHashesOf(t, documents.otherOfTheNodeListingAThousand),
	)
	nodepeer.PushURLMetadataRows(
		t, ctx, probe, nodeURL, nodeHash,
		urlMetadataRowsOf(t, documents.otherOfTheNodeListingAThousand),
	)
}

func peerHashAtTheSecondWordIn(t *testing.T, partition uint) yacymodel.Hash {
	t.Helper()

	return yacymodel.HashFromDHTRingPosition(positionOfTheSecondWordIn(t, partition))
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

	nodeHash := peerHashAtTheSecondWordIn(t, firstPartition)
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: networkName,
		Alias:       judgementHolderAlias,
		Hash:        nodeHash,
		SeedlistURL: seedlistURL,
	})

	nodepeer.PushPostings(
		t, ctx, probe, nodeURL, nodeHash,
		yacymodel.WordHash(judgementFirstWordToken),
		urlHashesOf(t, documents.named),
	)
	nodepeer.PushPostings(
		t, ctx, probe, nodeURL, nodeHash,
		yacymodel.WordHash(judgementSecondWordToken),
		urlHashesOf(t, slices.Concat(
			documents.named[:amountOfNamedDocumentsOnTheHolder],
			documents.otherOfTheHolder,
		)),
	)
	nodepeer.PushURLMetadataRows(
		t, ctx, probe, nodeURL, nodeHash,
		urlMetadataRowsOf(t, slices.Concat(documents.named, documents.otherOfTheHolder)),
	)
}

func judgementSettings() map[string]string {
	return map[string]string{
		"YACYDHTSEARCH_PARTITION_EXPONENT":   strconv.Itoa(partitionExponent),
		"YACYDHTSEARCH_HEDGE_DELAY":          judgementHedgeDelay.String(),
		"YACYDHTSEARCH_RANKING_LIFETIME":     judgementRankingLifetime.String(),
		"YACYDHTSEARCH_RANKED_ITEMS_CEILING": strconv.Itoa(amountOfNamedDocuments),
		"YACYDHTSEARCH_REFRESH_INTERVAL":     directoryRefreshInterval.String(),
	}
}

func waitForEveryPeerToAnswer(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	service yacydhtsearchService,
	peerName string,
) {
	t.Helper()

	const everyPeerAnswers = "yacydhtsearch_directory_answering_peers 3"
	if metricLineShowedUp(t, ctx, probe, service, everyPeerAnswers, answeringPeersTimeout) {
		return
	}
	t.Fatalf(
		"%s never held the holder and the two peers beside %s as answering peers:\n%s",
		judgementSearchAlias,
		peerName,
		publishedBy(t, ctx, probe, service),
	)
}

func metricLineShowedUp(
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

	if metricLineShowedUp(t, ctx, probe, service, line, judgementTimeout) {
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
		`yacydhtsearch_peer_judgements_total{judged=%q,question=%q} %d`,
		judged,
		judgementQuestion,
		counted,
	)
}

func requireMoreNamedDocumentsThanTheHolderLists(
	t *testing.T,
	peerName string,
	links, named []string,
) {
	t.Helper()

	answered := amountOfNamedDocumentsIn(links, named)
	if answered > amountOfNamedDocumentsOnTheHolder {
		return
	}
	t.Fatalf(
		"%s left the results at %d of the %d named documents, want more than the %d the holder"+
			" lists for both words; %d results came back",
		peerName,
		answered,
		len(named),
		amountOfNamedDocumentsOnTheHolder,
		len(links),
	)
}

func amountOfNamedDocumentsIn(links, named []string) int {
	answered := 0
	for _, document := range named {
		if slices.Contains(links, document) {
			answered++
		}
	}

	return answered
}

func standingsLine(standing string, counted int) string {
	return fmt.Sprintf(
		`yacydhtsearch_peer_standings_total{question=%q,standing=%q} %d`,
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
