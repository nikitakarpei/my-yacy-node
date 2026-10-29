//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"io"
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
	httpsPreferringYaCyAlias = "yacy-https-e2e"
	httpsPreferringNodeAlias = "node-https-e2e"
	secureYaCyAlias          = "yacy-secure-e2e"
	callingNodeAlias         = "node-calling-e2e"
	secureNodeAlias          = "node-secure-e2e"
	closedSecurePortAlias    = "node-closed-secure-port-e2e"
	closedSecurePort         = "9"
)

func TestJavaPeerThatPrefersHTTPSKeepsTheSecurePortOfTheNode(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)
	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)
	yacy, yacyURL := yacypeer.Start(
		t,
		ctx,
		probe,
		network.Name,
		httpsPreferringYaCyAlias,
		"sed -i 's#^network.unit.protocol.https.preferred.*#network.unit.protocol.https.preferred = true#' "+
			"/opt/yacy_search_server/defaults/yacy.init",
		"sed -i 's#^remotesearch.https.preferred.*#remotesearch.https.preferred = true#' "+
			"/opt/yacy_search_server/defaults/yacy.init",
	)
	nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       httpsPreferringNodeAlias,
		Key:         httpsPreferringNodeKey,
		SeedlistURL: "http://" + httpsPreferringYaCyAlias + ":" + peerclient.Port + "/yacy/seedlist.html",
	})
	nodeHash := httpsPreferringNodeKey.Hash()
	waitPeerActiveConnected(t, ctx, probe, yacyURL, nodeHash, 60*time.Second)

	contactedOverHTTPS := pollwait.For(60*time.Second, func() bool {
		return containerLogMentions(
			t,
			ctx,
			yacy,
			"contacted peer at https://"+httpsPreferringNodeAlias+":8443/",
		)
	})
	if !contactedOverHTTPS {
		t.Fatalf("Java never contacted %s at its secure port", nodeHash)
	}

	seed, listed := javaSeedOf(t, ctx, probe, yacyURL, nodeHash)
	capabilities, _ := seed.Capabilities.Get()
	if !listed || !capabilities.SSLAvailable ||
		seed.SecurePort != yacymodel.Some(yacymodel.Port(8443)) {
		t.Fatalf("Java lists %s with SSL %v at secure port %v (listed %v), want SSL kept at 8443",
			nodeHash, capabilities.SSLAvailable, seed.SecurePort, listed)
	}
}

func TestNodeReachesASignedPeerOnlyAtItsSecurePort(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)
	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)
	yacypeer.Start(t, ctx, probe, network.Name, secureYaCyAlias)
	seedlistURL := "http://" + secureYaCyAlias + ":" + peerclient.Port + "/yacy/seedlist.html"
	callingNode, callingURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       callingNodeAlias,
		Key:         callingNodeKey,
		SeedlistURL: seedlistURL,
	})
	nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       secureNodeAlias,
		Key:         secureNodeKey,
		SeedlistURL: seedlistURL,
	})
	nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName:          network.Name,
		Alias:                closedSecurePortAlias,
		Key:                  closedSecurePortNodeKey,
		SeedlistURL:          seedlistURL,
		AdvertisedSecurePort: closedSecurePort,
	})

	reached := pollwait.For(120*time.Second, func() bool {
		_, known := knownSeedOf(t, ctx, probe, callingURL, secureNodeKey.Hash())
		return known
	})
	if !reached {
		t.Fatalf("calling node never reached %s at its secure port", secureNodeKey.Hash())
	}
	closedSecurePortHash := closedSecurePortNodeKey.Hash().String()
	greetFailed := pollwait.For(60*time.Second, func() bool {
		return containerLogMentions(t, ctx, callingNode,
			`"msg":"peer greet failed","peer":"`+closedSecurePortHash+`"`)
	})
	if !greetFailed {
		t.Fatalf("calling node never tried %s", closedSecurePortHash)
	}

	if _, known := knownSeedOf(t, ctx, probe, callingURL, closedSecurePortNodeKey.Hash()); known {
		t.Fatalf(
			"calling node reached %s, whose secure port is closed",
			closedSecurePortNodeKey.Hash(),
		)
	}
}

func containerLogMentions(
	t *testing.T,
	ctx context.Context,
	container testcontainers.Container,
	text string,
) bool {
	t.Helper()
	logs, err := container.Logs(ctx)
	if err != nil {
		return false
	}
	defer func() { _ = logs.Close() }()
	written, err := io.ReadAll(logs)
	return err == nil && bytes.Contains(written, []byte(text))
}

func javaSeedOf(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	peer yacymodel.Hash,
) (yacymodel.Seed, bool) {
	t.Helper()
	result := probe.Get(ctx, yacyURL+"/yacy/seedlist.html?id="+peer.String())
	if !result.OK {
		return yacymodel.Seed{}, false
	}
	seedList, err := yacyproto.ParseSeedListResponse(ctx, bytes.NewReader([]byte(result.Body)))
	if err != nil || len(seedList.Seeds) != 1 {
		return yacymodel.Seed{}, false
	}
	return seedList.Seeds[0], true
}
