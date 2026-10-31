//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/yacypeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	provenPostingYaCyAlias   = "yacy-proven-e2e"
	offeringNodeAlias        = "node-offering-e2e"
	receivingNodeAlias       = "node-receiving-e2e"
	crawlerPartitionExponent = 4

	reachablePeers            = "yacynode_peerroster_reachable_peers"
	peersAcceptingRemoteIndex = "yacynode_rwidistribution_peers_accepting_remote_index"
	distributionMetrics       = "yacynode_rwidistribution_"
	postingsDroppedUnrooted   = `yacynode_remoteindexadmission_received_postings_total{outcome="dropped_unrooted_version"}`
)

var (
	provenWords    = []yacymodel.Hash{mustHash("PROVENWORD01"), mustHash("PROVENWORD02")}
	unprovenWord   = mustHash("PROVENWORD03")
	provenDocument = yacymodel.URLMetadata{
		Hash:    mustURLHash("http://example.invalid/proven-e2e"),
		Address: "http://example.invalid/proven-e2e",
		Title:   "proven page",
	}
)

func TestGoPeerIndexesOfferedProvenPostingsAndDropsAnUnprovenOneOfTheirPage(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)
	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)
	_, yacyURL := yacypeer.Start(t, ctx, probe, network.Name, provenPostingYaCyAlias)
	seedlistURL := "http://" + provenPostingYaCyAlias + ":" + peerclient.Port + "/yacy/seedlist.html"
	receivingNode, receivingURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       receivingNodeAlias,
		Key:         receivingNodeKey,
		SeedlistURL: seedlistURL,
	})
	offeringNode, offeringURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       offeringNodeAlias,
		Key:         offeringNodeKey,
		SeedlistURL: seedlistURL,
		Distribution: nodepeer.DistributionConfig{
			Enabled:           true,
			Redundancy:        3,
			PartitionExponent: 1,
			PostingsPerBatch:  10,
			RefreshInterval:   5 * time.Second,
			MinReachablePeers: 1,
		},
	})
	waitPeerActiveConnected(t, ctx, probe, yacyURL, receivingNodeKey.Hash(), 60*time.Second)
	waitPeerActiveConnected(t, ctx, probe, yacyURL, offeringNodeKey.Hash(), 60*time.Second)
	waitRingOfBothPeers(t, ctx, probe, offeringNode)

	crawlerPartitions, err := yacymodel.DHTRingPartitionsFromExponent(crawlerPartitionExponent)
	if err != nil {
		t.Fatalf("crawler partitions: %v", err)
	}
	crawled := nodepeer.CrawledPagePostingsFrom(t, provenDocument, provenWords, crawlerPartitions)
	rootedDocument := provenDocument
	rootedDocument.PostingsRoot = crawled.Root()
	nodepeer.PushURLMetadata(t, ctx, probe, offeringURL, offeringNodeKey.Hash(), rootedDocument)
	nodepeer.PushPagePostings(t, ctx, probe, offeringURL, offeringNodeKey.Hash(), crawled)
	waitHeldPostings(t, ctx, probe, offeringURL, offeringNodeKey.Hash(), len(provenWords))
	if !heldPostingsReach(ctx, probe, receivingURL, receivingNodeKey.Hash(), len(provenWords)) {
		dumpDistributionMetrics(t, ctx, probe, offeringNode)
		t.Fatalf("receiving node never held the %d offered postings", len(provenWords))
	}

	nodepeer.PushPosting(
		t,
		ctx,
		probe,
		receivingURL,
		receivingNodeKey.Hash(),
		unprovenWord,
		provenDocument,
	)

	if dropped, _ := nodepeer.SeriesValueOf(
		t, ctx, probe, receivingNode, postingsDroppedUnrooted,
	); dropped != 1 {
		t.Fatalf("receiving node dropped %v unproven postings of the rooted page, want 1", dropped)
	}
}

func waitRingOfBothPeers(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	node testcontainers.Container,
) {
	t.Helper()
	if pollwait.For(60*time.Second, func() bool {
		return nodepeer.SeriesSumOf(t, ctx, probe, node, reachablePeers) >= 2 &&
			nodepeer.SeriesSumOf(t, ctx, probe, node, peersAcceptingRemoteIndex) >= 2
	}) {
		return
	}
	dumpDistributionMetrics(t, ctx, probe, node)
	t.Fatal("offering node never reached both peers on its DHT ring")
}

func waitHeldPostings(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	nodeURL string,
	nodeHash yacymodel.Hash,
	want int,
) {
	t.Helper()
	if !heldPostingsReach(ctx, probe, nodeURL, nodeHash, want) {
		t.Fatalf("node %s never held %d postings", nodeHash, want)
	}
}

func heldPostingsReach(
	ctx context.Context,
	probe *httpprobe.Probe,
	nodeURL string,
	nodeHash yacymodel.Hash,
	want int,
) bool {
	return pollwait.For(120*time.Second, func() bool {
		count, ok := peerclient.QueryCount(ctx, probe, nodeURL, nodeHash, yacyproto.ObjectRWICount)

		return ok && count >= want
	})
}

func dumpDistributionMetrics(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	node testcontainers.Container,
) {
	t.Helper()
	for _, line := range nodepeer.MetricLinesOf(t, ctx, probe, node, reachablePeers) {
		t.Log(line)
	}
	for _, line := range nodepeer.MetricLinesOf(t, ctx, probe, node, distributionMetrics) {
		t.Log(line)
	}
}
