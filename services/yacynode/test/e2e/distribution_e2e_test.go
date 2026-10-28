//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/yacypeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
)

const (
	distributionYaCyAlias = "yacy-dist-e2e"
	distributionNodeAlias = "node-dist-e2e"
)

var (
	distributionWordHash = mustHash("DISTWORDHASH")
	distributionDocument = yacymodel.URLMetadata{
		Hash:    mustURLHash("http://example.invalid/dist-e2e"),
		Address: "http://example.invalid/dist-e2e",
	}
)

func TestNodeDistributesRWIToRealYaCy(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)

	egressproxy.Start(t, ctx, network.Name)

	_, yacyURL := yacypeer.Start(t, ctx, probe, network.Name, distributionYaCyAlias)
	yacyHash := peerclient.ResolveHash(t, ctx, probe, yacyURL)

	seedlistURL := "http://" + distributionYaCyAlias + ":" + peerclient.Port + "/yacy/seedlist.html"
	nodeKey := peerkeytest.Key(44)
	nodeHash := nodeKey.Identity().Hash()
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       distributionNodeAlias,
		Key:         nodeKey,
		SeedlistURL: seedlistURL,
		Distribution: nodepeer.DistributionConfig{
			Enabled:           true,
			Redundancy:        2,
			PartitionExponent: 1,
			PostingsPerBatch:  10,
			RefreshInterval:   5 * time.Second,
			MinReachablePeers: 1,
		},
	})

	waitPeerActiveConnected(t, ctx, probe, yacyURL, nodeHash, 60*time.Second)

	nodepeer.PushPosting(
		t,
		ctx,
		probe,
		nodeURL,
		nodeHash,
		distributionWordHash,
		distributionDocument,
	)

	nodepeer.PushURLMetadata(t, ctx, probe, nodeURL, nodeHash, distributionDocument)

	yacypeer.WaitRWICount(t, ctx, probe, yacyURL, yacyHash, 1, 60*time.Second)
}
