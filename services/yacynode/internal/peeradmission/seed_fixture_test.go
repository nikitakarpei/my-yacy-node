package peeradmission

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
)

const hashFiller = "AAAAAAAAAAAA"

func hashFor(base string) yacymodel.Hash {
	padded := base + hashFiller
	hash, err := yacymodel.ParseHash(padded[:yacymodel.HashLength])
	if err != nil {
		panic(err)
	}

	return hash
}

func callerSeed(t testing.TB, hash, ip string, port int) yacymodel.Seed {
	seed := yacymodel.Seed{Hash: hashFor(hash)}
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
		seed:        callerSeed(t, "self", "203.0.113.9", 8090),
	}
}
