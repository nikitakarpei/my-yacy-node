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
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	bindingYaCyAlias   = "yacy-binding-e2e"
	verifyingNodeAlias = "node-verifying-e2e"
	boundNodeAlias     = "node-bound-e2e"
	forgedHostText     = "198.51.100.9"
)

func TestNodeKeepsTheBoundIdentityOfAPeerAgainstAForgedHello(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)
	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)
	yacypeer.Start(t, ctx, probe, network.Name, bindingYaCyAlias)
	seedlistURL := "http://" + bindingYaCyAlias + ":" + peerclient.Port + "/yacy/seedlist.html"
	_, verifyingURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       verifyingNodeAlias,
		Key:         verifyingNodeKey,
		SeedlistURL: seedlistURL,
	})
	nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       boundNodeAlias,
		Key:         boundNodeKey,
		SeedlistURL: seedlistURL,
	})
	boundHash := boundNodeKey.Hash()

	bound := pollwait.For(120*time.Second, func() bool {
		seed, known := knownSeedOf(t, ctx, probe, verifyingURL, boundHash)
		identity, identified := seed.Identity.Get()
		return known && identified && identity == boundNodeKey.Identity()
	})
	if !bound {
		t.Fatalf("verifying node never listed %s with its identity", boundHash)
	}

	peerclient.Hello(t, ctx, probe, verifyingURL, forgedSeedOf(t, boundHash))

	seed, known := knownSeedOf(t, ctx, probe, verifyingURL, boundHash)
	if !known {
		t.Fatalf("verifying node no longer lists %s", boundHash)
	}
	for _, host := range seed.AdditionalHosts.OrElse(nil) {
		if host.String() == forgedHostText {
			t.Fatalf("verifying node lists the forged host %s for %s", forgedHostText, boundHash)
		}
	}
}

func knownSeedOf(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	peerURL string,
	peer yacymodel.Hash,
) (yacymodel.Seed, bool) {
	t.Helper()
	for _, seed := range peerclient.Hello(t, ctx, probe, peerURL, unaddressedCallerSeed(t)).KnownSeeds() {
		if seed.Hash == peer {
			return seed, true
		}
	}
	return yacymodel.Seed{}, false
}

func unaddressedCallerSeed(t *testing.T) yacymodel.Seed {
	t.Helper()
	return yacymodel.Seed{
		Hash:     mustHash("BINDPROBE001"),
		Name:     mustPeerName(t, "binding-probe"),
		PeerType: yacymodel.PeerJunior,
	}
}

func forgedSeedOf(t *testing.T, peer yacymodel.Hash) yacymodel.Seed {
	t.Helper()
	port, err := yacymodel.ParsePort(peerclient.Port)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return yacymodel.Seed{
		Hash:            peer,
		Name:            mustPeerName(t, "forged-peer"),
		PeerType:        yacymodel.PeerSenior,
		PrimaryHost:     yacymodel.Some(mustHost(t, boundNodeAlias)),
		AdditionalHosts: yacymodel.Some([]yacymodel.Host{mustHost(t, forgedHostText)}),
		Port:            yacymodel.Some(port),
	}
}

func mustPeerName(t *testing.T, text string) yacymodel.PeerName {
	t.Helper()
	name, err := yacymodel.ParsePeerName(text)
	if err != nil {
		t.Fatalf("parse peer name %q: %v", text, err)
	}
	return name
}

func mustHost(t *testing.T, text string) yacymodel.Host {
	t.Helper()
	host, err := yacymodel.ParseHost(text)
	if err != nil {
		t.Fatalf("parse host %q: %v", text, err)
	}
	return host
}
