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
	judgementFirstWordHolderAlias = "yacy-named-documents-first-word-e2e"
	nodeUnderJudgementAlias       = "node-named-documents-e2e"
	yacyPeerUnderJudgementAlias   = "yacy-named-documents-e2e"
	judgementSearchAlias          = "yacydhtsearch-named-documents"

	nodeUnderJudgementHash = "JUDGEDNODE01"

	namedDocumentsForm = "named documents"

	judgementFirstWordToken  = "yacydhtsearchnameddocumentsfirstword"
	judgementSecondWordToken = "yacydhtsearchnameddocumentssecondword"
	judgementDocumentTitle   = "named documents probe"

	// documentsOfTheSecondWord is above the thousand documents a peer lists for
	// one word, so the peer under judgement always leaves some of them unlisted.
	documentsOfTheSecondWord = 1200
	namedDocuments           = 200
	documentsPerPush         = 200

	judgementRankingLifetime = time.Second
	pauseBetweenTheQueries   = 5 * time.Second
	directoryRefreshInterval = 10 * time.Second
	answeringPeersTimeout    = 5 * time.Minute
	judgementTimeout         = 60 * time.Second
)

func TestANodeThatHonorsTheNamedDocumentsIsAskedAgain(t *testing.T) {
	judgeTheSecondWordHolder(t, peerUnderJudgement{
		name:                nodeUnderJudgementAlias,
		holdTheSecondWord:   nodeHoldingTheSecondWord,
		judged:              "honored",
		standing:            "honoring",
		answeredCrossChecks: 2,
		requireResults:      requireEveryNamedDocument,
	})
}

func TestAYaCyPeerThatIgnoresTheNamedDocumentsIsNotAskedAgain(t *testing.T) {
	judgeTheSecondWordHolder(t, peerUnderJudgement{
		name:                 yacyPeerUnderJudgementAlias,
		holdTheSecondWord:    yacyPeerHoldingTheSecondWord,
		seedlistURLsNamingIt: []string{seedlistURLOf(yacyPeerUnderJudgementAlias)},
		judged:               "ignored",
		standing:             "ignoring",
		answeredCrossChecks:  1,
		requireResults:       requireFewerThanTheNamedDocuments,
	})
}

type peerUnderJudgement struct {
	name                 string
	holdTheSecondWord    secondWordHolderStart
	seedlistURLsNamingIt []string
	judged               string
	standing             string
	answeredCrossChecks  int
	requireResults       resultsRequirement
}

type secondWordHolderStart func(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	bootstrapSeedlistURL string,
	documents secondWordDocuments,
)

type secondWordDocuments struct {
	ofTheSecondWordOnly []string
	alsoOfTheFirstWord  []string
}

type resultsRequirement func(t *testing.T, peerName string, links, named []string)

func judgeTheSecondWordHolder(t *testing.T, peer peerUnderJudgement) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)

	documents := secondWordDocumentsInURLHashOrder(t)
	startFirstWordHolder(t, ctx, probe, network.Name, documents.alsoOfTheFirstWord)
	firstWordSeedlistURL := seedlistURLOf(judgementFirstWordHolderAlias)
	peer.holdTheSecondWord(t, ctx, probe, network.Name, firstWordSeedlistURL, documents)

	service := startYacydhtsearch(
		t,
		ctx,
		network.Name,
		judgementSearchAlias,
		strings.Join(append([]string{firstWordSeedlistURL}, peer.seedlistURLsNamingIt...), ","),
		judgementSettings(),
	)
	waitForBothPeersToAnswer(t, ctx, probe, service)

	query := judgementFirstWordToken + " " + judgementSecondWordToken
	links := resultLinksFor(t, ctx, probe, service.searchURL, query, namedDocuments)
	waitForMetricLine(t, ctx, probe, service, judgementsLine(peer.judged, 1), peer.name)
	requireMetricLine(t, ctx, probe, service, standingsLine("never judged", 1), peer.name)
	peer.requireResults(t, peer.name, links, documents.alsoOfTheFirstWord)

	time.Sleep(pauseBetweenTheQueries)
	resultLinksFor(t, ctx, probe, service.searchURL, query, namedDocuments)
	waitForMetricLine(t, ctx, probe, service, standingsLine(peer.standing, 1), peer.name)
	requireMetricLine(
		t, ctx, probe, service, answeredCrossChecksLine(peer.answeredCrossChecks), peer.name,
	)
}

// secondWordDocumentsInURLHashOrder names the documents of the second word in
// the order a peer lists them in, so that the documents the first word holder
// also holds are the ones a listing cut at a thousand leaves out.
func secondWordDocumentsInURLHashOrder(t *testing.T) secondWordDocuments {
	t.Helper()

	addresses := make([]string, documentsOfTheSecondWord)
	urlHashes := make(map[string]string, documentsOfTheSecondWord)
	for i := range addresses {
		addresses[i] = fmt.Sprintf("http://transfer.example.invalid/judged-%04d.html", i)
		urlHashes[addresses[i]] = urlHashOf(t, addresses[i]).String()
	}
	slices.SortFunc(addresses, func(one, other string) int {
		return yacymodel.CompareInAlphabetOrder(urlHashes[one], urlHashes[other])
	})
	amountOfDocumentsListed := len(addresses) - namedDocuments

	return secondWordDocuments{
		ofTheSecondWordOnly: addresses[:amountOfDocumentsListed],
		alsoOfTheFirstWord:  addresses[amountOfDocumentsListed:],
	}
}

func urlHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	urlHash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("url hash of %s: %v", address, err)
	}

	return urlHash
}

func startFirstWordHolder(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	documents []string,
) {
	t.Helper()

	_, holderURL := yacypeer.Start(
		t, ctx, probe, networkName, judgementFirstWordHolderAlias,
		yacypeer.RemoteSearchOverrides()...,
	)
	pushInBatches(t, ctx, probe, holderURL, documents, []string{judgementFirstWordToken})
}

func pushInBatches(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	documentAddresses []string,
	tokens []string,
) {
	t.Helper()

	for batch := range slices.Chunk(documentAddresses, documentsPerPush) {
		yacypeer.PushDocumentsUnderAddresses(t, ctx, probe, yacyURL, batch, tokens)
	}
}

func judgementSettings() map[string]string {
	return map[string]string{
		"YACYDHTSEARCH_RANKING_LIFETIME":                 judgementRankingLifetime.String(),
		"YACYDHTSEARCH_ASKS_FOR_CROSS_CHECKED_DOCUMENTS": "true",
		"YACYDHTSEARCH_RANKED_ITEMS_CEILING":             strconv.Itoa(namedDocuments),
		"YACYDHTSEARCH_REFRESH_INTERVAL":                 directoryRefreshInterval.String(),
	}
}

func waitForBothPeersToAnswer(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	service yacydhtsearchService,
) {
	t.Helper()

	const bothPeersAnswer = "yacydhtsearch_directory_answering_peers 2"
	if metricLineShowedUp(t, ctx, probe, service, bothPeersAnswer, answeringPeersTimeout) {
		return
	}
	t.Fatalf(
		"%s never held the first word holder and the peer under judgement as answering peers:\n%s",
		judgementSearchAlias,
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

func judgementsLine(judged string, counted int) string {
	return fmt.Sprintf(
		`yacydhtsearch_peer_judgements_total{form=%q,judged=%q} %d`,
		namedDocumentsForm,
		judged,
		counted,
	)
}

func standingsLine(standing string, counted int) string {
	return fmt.Sprintf(
		`yacydhtsearch_peer_standings_total{form=%q,standing=%q} %d`,
		namedDocumentsForm,
		standing,
		counted,
	)
}

func answeredCrossChecksLine(counted int) string {
	return fmt.Sprintf(
		`yacydhtsearch_peer_calls_total{asked_for="cross-checked documents",outcome="answered"} %d`,
		counted,
	)
}

func requireEveryNamedDocument(t *testing.T, peerName string, links, named []string) {
	t.Helper()

	answered := amountOfNamedDocumentsIn(links, named)
	if answered == len(named) {
		return
	}
	t.Fatalf(
		"%s answered %d of the %d named documents, want every one of them; %d results came back",
		peerName,
		answered,
		len(named),
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

func requireFewerThanTheNamedDocuments(t *testing.T, peerName string, links, named []string) {
	t.Helper()

	answered := amountOfNamedDocumentsIn(links, named)
	if answered < len(named) {
		return
	}
	t.Fatalf(
		"%s answered every one of the %d named documents, want fewer because it lists its own thousand",
		peerName,
		len(named),
	)
}

// nodeHoldingTheSecondWord starts the node under judgement on the seedlist of
// the first word holder, which then names the node in the seedlist the service
// reads, and feeds the node the second word over every document in one call.
func nodeHoldingTheSecondWord(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	bootstrapSeedlistURL string,
	documents secondWordDocuments,
) {
	t.Helper()

	nodeHash := peerHashOf(t, nodeUnderJudgementHash)
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: networkName,
		Alias:       nodeUnderJudgementAlias,
		Hash:        nodeHash,
		SeedlistURL: bootstrapSeedlistURL,
	})

	addresses := slices.Concat(documents.ofTheSecondWordOnly, documents.alsoOfTheFirstWord)
	nodepeer.PushPostings(
		t, ctx, probe, nodeURL, nodeHash,
		yacymodel.WordHash(judgementSecondWordToken),
		urlHashesOf(t, addresses),
	)
	nodepeer.PushURLMetadataRows(t, ctx, probe, nodeURL, nodeHash, urlMetadataRowsOf(t, addresses))
}

func peerHashOf(t *testing.T, raw string) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(raw)
	if err != nil {
		t.Fatalf("peer hash %s: %v", raw, err)
	}

	return hash
}

func urlHashesOf(t *testing.T, addresses []string) []yacymodel.URLHash {
	t.Helper()

	urlHashes := make([]yacymodel.URLHash, 0, len(addresses))
	for _, address := range addresses {
		urlHashes = append(urlHashes, urlHashOf(t, address))
	}

	return urlHashes
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

// yacyPeerHoldingTheSecondWord starts the YaCy peer under judgement, which the
// service reads from that peer's own seedlist, and pushes it every document of
// the second word, the named ones with the first word as well.
func yacyPeerHoldingTheSecondWord(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName string,
	_ string,
	documents secondWordDocuments,
) {
	t.Helper()

	_, yacyURL := yacypeer.Start(
		t, ctx, probe, networkName, yacyPeerUnderJudgementAlias,
		yacypeer.RemoteSearchOverrides()...,
	)
	pushInBatches(
		t, ctx, probe, yacyURL, documents.ofTheSecondWordOnly,
		[]string{judgementSecondWordToken},
	)
	pushInBatches(
		t, ctx, probe, yacyURL, documents.alsoOfTheFirstWord,
		[]string{judgementFirstWordToken, judgementSecondWordToken},
	)
}
