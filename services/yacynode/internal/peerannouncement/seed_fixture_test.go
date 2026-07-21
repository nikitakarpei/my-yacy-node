package peerannouncement

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

const seedPort = 8090

func callerSeed(t testing.TB, hash, ip string) yacymodel.Seed {
	t.Helper()

	host, err := yacymodel.ParseHost(ip)
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	name, err := yacymodel.ParsePeerName(hashFor(hash).String())
	if err != nil {
		t.Fatalf("parse peer name: %v", err)
	}

	return yacymodel.Seed{
		Hash:           hashFor(hash),
		Name:           name,
		PeerType:       yacymodel.PeerSenior,
		PrimaryAddress: yacymodel.Some(host),
		Port:           yacymodel.Some(yacymodel.Port(seedPort)),
	}
}
