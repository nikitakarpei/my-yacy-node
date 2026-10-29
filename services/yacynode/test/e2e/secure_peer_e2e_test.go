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
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/yacypeer"
)

const (
	secureYaCyAlias       = "yacy-secure-e2e"
	callingNodeAlias      = "node-calling-e2e"
	secureNodeAlias       = "node-secure-e2e"
	closedSecurePortAlias = "node-closed-secure-port-e2e"
	closedPort            = "9"
)

const (
	peersReachedOverHTTP  = `yacynode_peerannouncement_reachable_peers_per_scheme{scheme="http"}`
	peersReachedOverHTTPS = `yacynode_peerannouncement_reachable_peers_per_scheme{scheme="https"}`
	peersThatDidNotAnswer = `yacynode_peerannouncement_peers_per_contact_outcome{outcome="did_not_answer"}`
)

func TestNodeReachesASignedPeerOnlyAtItsSecurePort(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)
	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)
	yacypeer.Start(t, ctx, probe, network.Name, secureYaCyAlias)
	seedlistURL := "http://" + secureYaCyAlias + ":" + peerclient.Port + "/yacy/seedlist.html"
	callingNode, _ := nodepeer.Start(t, ctx, probe, nodepeer.Config{
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
		AdvertisedSecurePort: closedPort,
	})

	valueOf := func(series string) float64 {
		value, _ := nodepeer.SeriesValueOf(t, ctx, probe, callingNode, series)
		return value
	}
	expectedRoundSeen := pollwait.For(
		120*time.Second,
		func() bool {
			return valueOf(peersReachedOverHTTP) == 1 &&
				valueOf(peersReachedOverHTTPS) == 1 &&
				valueOf(peersThatDidNotAnswer) == 1
		},
	)
	if !expectedRoundSeen {
		t.Fatalf(
			"calling node never had a round with Java over HTTP, %s over HTTPS and %s unanswered; "+
				"last round: http=%v https=%v did_not_answer=%v",
			secureNodeAlias,
			closedSecurePortAlias,
			valueOf(
				peersReachedOverHTTP,
			),
			valueOf(peersReachedOverHTTPS),
			valueOf(peersThatDidNotAnswer),
		)
	}
}
