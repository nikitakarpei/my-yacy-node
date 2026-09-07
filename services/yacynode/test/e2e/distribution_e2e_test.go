//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/yacypeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	distributionYaCyAlias = "yacy-dist-e2e"
	distributionNodeAlias = "node-dist-e2e"

	escrowHoldGrace = 15 * time.Second
)

var (
	distributionWordHash = mustHash("DISTWORDHASH")
	distributionDocument = yacymodel.URLMetadata{Address: "http://example.invalid/dist-e2e"}
)

func TestNodeDistributesRWIToRealYaCy(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)

	egressproxy.Start(t, ctx, network.Name)

	_, yacyURL := yacypeer.Start(t, ctx, probe, network.Name, distributionYaCyAlias)
	yacyHash := peerclient.ResolveHash(t, ctx, probe, yacyURL)

	seedlistURL := "http://" + distributionYaCyAlias + ":" + peerclient.Port + "/yacy/seedlist.html"
	nodeHash := mustHash("DISTNODEHASH")
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       distributionNodeAlias,
		Hash:        nodeHash,
		SeedlistURL: seedlistURL,
		Distribution: nodepeer.DistributionConfig{
			Enabled:               true,
			Redundancy:            2,
			PartitionExponent:     1,
			PostingsPerBatch:      10,
			CycleInterval:         time.Second,
			DrainBudget:           time.Second,
			LongestOfferInterval:  5 * time.Second,
			ShortestOfferInterval: 2 * time.Second,
			MinReachablePeers:     1,
		},
	})

	waitPeerActiveConnected(t, ctx, probe, yacyURL, nodeHash, 60*time.Second)

	documentHash, err := distributionDocument.Hash()
	if err != nil {
		t.Fatalf("document hash: %v", err)
	}

	nodepeer.PushPosting(t, ctx, probe, nodeURL, nodeHash, distributionWordHash, documentHash)

	if pollwait.For(escrowHoldGrace, func() bool {
		count, ok := peerclient.QueryCount(ctx, probe, yacyURL, yacyHash, yacyproto.ObjectRWICount)

		return ok && count > 0
	}) {
		t.Fatal("node distributed a posting before its url metadata arrived")
	}

	nodepeer.PushURLMetadata(t, ctx, probe, nodeURL, nodeHash, distributionDocument)

	yacypeer.WaitRWICount(t, ctx, probe, yacyURL, yacyHash, 1, 60*time.Second)
}
