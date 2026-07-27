//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/yacypeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	distributionYaCyAlias = "yacy-dist-e2e"
	distributionNodeAlias = "node-dist-e2e"
)

var (
	distributionWordHash = mustHash("DISTWORDHASH")
	distributionDocHash  = mustHash("DISTDOCHASH1")
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
		},
	})

	waitNodeSeenActive(t, ctx, probe, yacyURL, nodeHash, 60*time.Second)

	nodepeer.PushPosting(
		t, ctx, probe, nodeURL, nodeHash, distributionWordHash, distributionDocHash,
	)

	waitYaCyRWICount(t, ctx, probe, yacyURL, yacyHash, 1, 60*time.Second,
		"real YaCy never received the distributed posting")
}

func waitNodeSeenActive(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	nodeHash yacymodel.Hash,
	timeout time.Duration,
) {
	t.Helper()
	if pollwait.For(timeout, func() bool {
		result := probe.Get(ctx, yacyURL+"/Network.xml?page=1&maxCount=1000")
		if !result.OK {
			return false
		}
		active, err := peerdirectory.ActivePeerHashes([]byte(result.Body))
		if err != nil {
			return false
		}
		_, ok := active[nodeHash.String()]
		return ok
	}) {
		return
	}
	t.Fatal("real YaCy never saw the node as an active connected peer")
}

func waitYaCyRWICount(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	yacyHash yacymodel.Hash,
	want int,
	timeout time.Duration,
	failMessage string,
) {
	t.Helper()
	last := -1
	if pollwait.For(timeout, func() bool {
		count, ok := peerclient.QueryCount(ctx, probe, yacyURL, yacyHash, yacyproto.ObjectRWICount)
		if !ok {
			return false
		}
		last = count
		return count >= want
	}) {
		return
	}
	t.Fatalf("%s (last rwicount=%d)", failMessage, last)
}
