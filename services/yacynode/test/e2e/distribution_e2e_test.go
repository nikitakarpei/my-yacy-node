//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/yacypeer"
)

const (
	distributionYaCyAlias = "yacy-dist-e2e"
	distributionNodeAlias = "node-dist-e2e"
)

var (
	distributionWordHash = mustHash("DISTWORDHASH")
	distributionDocHash  = mustURLHash("DISTDOCHASH1")
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
			Enabled:           true,
			Redundancy:        1,
			PartitionExponent: 1,
			PostingsPerCycle:  10,
			CycleInterval:     time.Second,
			RefreshInterval:   5 * time.Second,
			RetryInterval:     2 * time.Second,
			MinReachablePeers: 1,
		},
	})

	waitPeerActiveConnected(t, ctx, probe, yacyURL, nodeHash, 60*time.Second)

	nodepeer.PushPosting(
		t, ctx, probe, nodeURL, nodeHash, distributionWordHash, distributionDocHash,
	)

	yacypeer.WaitRWICount(t, ctx, probe, yacyURL, yacyHash, 1, 60*time.Second)
}
