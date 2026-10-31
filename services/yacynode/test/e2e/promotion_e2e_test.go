//go:build e2e

package e2e

import (
	"context"
	"net/url"
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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/peerdirectory"
)

const (
	yacyAlias = "yacy-e2e"
	nodeAlias = "node-e2e"
)

var (
	nodeKey  = promotionNodeKey
	nodeHash = nodeKey.Hash()
)

func mustURLHash(address string) yacymodel.URLHash {
	webAddress, err := url.Parse(address)
	if err != nil {
		panic(err)
	}

	return yacymodel.URLHashOf(webAddress)
}

func mustHash(raw string) yacymodel.Hash {
	hash, err := yacymodel.ParseHash(raw)
	if err != nil {
		panic(err)
	}

	return hash
}

var httpsPreferringYaCyOverrides = []string{
	"sed -i 's#^network.unit.protocol.https.preferred.*#network.unit.protocol.https.preferred = true#' " +
		"/opt/yacy_search_server/defaults/yacy.init",
	"sed -i 's#^remotesearch.https.preferred.*#remotesearch.https.preferred = true#' " +
		"/opt/yacy_search_server/defaults/yacy.init",
}

const connectionsOverHTTPS = `yacynode_peerport_connections_total{scheme="https"}`

func TestRealYaCyPromotesNodeToSenior(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	nodePromotedBy(t, ctx, probe)
}

func TestRealYaCyThatPrefersHTTPSPromotesNodeOverHTTPS(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	node := nodePromotedBy(t, ctx, probe, httpsPreferringYaCyOverrides...)

	if connections, _ := nodepeer.SeriesValueOf(
		t,
		ctx,
		probe,
		node,
		connectionsOverHTTPS,
	); connections == 0 {
		t.Fatal("real YaCy promoted the node without a single HTTPS connection to it")
	}
}

func nodePromotedBy(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyOverrides ...string,
) testcontainers.Container {
	t.Helper()
	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)
	_, yacyURL := yacypeer.Start(t, ctx, probe, network.Name, yacyAlias, yacyOverrides...)
	node, _ := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       nodeAlias,
		Key:         nodeKey,
		SeedlistURL: "http://" + yacyAlias + ":" + peerclient.Port + "/yacy/seedlist.html",
	})

	waitPeerActiveConnected(t, ctx, probe, yacyURL, nodeHash, 15*time.Second)

	promoted := pollwait.For(45*time.Second, func() bool {
		result := probe.Get(ctx, yacyURL+"/yacy/seedlist.xml")
		if !result.OK {
			return false
		}
		seniors, err := peerdirectory.SeniorHashes([]byte(result.Body))
		if err != nil {
			return false
		}
		_, ok := seniors[nodeHash.String()]
		return ok
	})
	if !promoted {
		if result := probe.Get(ctx, yacyURL+"/yacy/seedlist.xml"); result.OK {
			t.Logf("final seedlist.xml:\n%s", result.Body)
		}
		t.Fatalf("real YaCy never published node hash %s as PeerType=senior", nodeHash)
	}

	return node
}
