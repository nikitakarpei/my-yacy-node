package peeradmission_test

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
)

const (
	hashFiller = "AAAAAAAAAAAA"
	callerHash = "caller"
)

func hashFor(base string) yacymodel.Hash {
	padded := base + hashFiller
	hash, err := yacymodel.ParseHash(padded[:yacymodel.HashLength])
	if err != nil {
		panic(err)
	}

	return hash
}

func peerNameFor(t testing.TB, base string) yacymodel.PeerName {
	t.Helper()

	name, err := yacymodel.ParsePeerName((base + "xxxxxxxxxxxxxxxxxxxx")[:20])
	if err != nil {
		t.Fatalf("parse peer name: %v", err)
	}

	return name
}

func peerSeed(t testing.TB, hash, ip string, port int) yacymodel.Seed {
	seed := yacymodel.Seed{
		Hash:     hashFor(hash),
		Name:     peerNameFor(t, hash),
		PeerType: yacymodel.PeerJunior,
	}
	if ip != "" {
		host, err := yacymodel.ParseHost(ip)
		if err != nil {
			t.Fatalf("parse host: %v", err)
		}
		seed.PrimaryAddress = yacymodel.Some(host)
	}
	if port != 0 {
		seed.Port = yacymodel.Some(yacymodel.Port(port))
	}

	return seed
}

func reachableCallerSeed(t testing.TB, rawURL string) yacymodel.Seed {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	host, portText, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("split server host: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse server port: %v", err)
	}

	return peerSeed(t, callerHash, host, port)
}

func unreachableCallerSeed(t testing.TB) yacymodel.Seed {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split closed port addr: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse closed port: %v", err)
	}

	return peerSeed(t, callerHash, host, port)
}

func addresslessCallerSeed(t testing.TB) yacymodel.Seed {
	t.Helper()

	return peerSeed(t, callerHash, "", 0)
}

type stubStatus struct {
	networkName string
	seed        yacymodel.Seed
}

func (s stubStatus) NetworkName(context.Context) string {
	return s.networkName
}

func (s stubStatus) SelfSeed(context.Context) yacymodel.Seed {
	return s.seed
}

func localPeer() nodeidentity.Identity {
	return nodeidentity.Identity{Hash: hashFor("self"), NetworkName: "freeworld"}
}

func selfStatus(t testing.TB) stubStatus {
	return stubStatus{
		networkName: "freeworld",
		seed:        peerSeed(t, "self", "203.0.113.9", 8090),
	}
}
