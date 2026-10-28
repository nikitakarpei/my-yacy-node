package yacymodel_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
)

const signedNetworkName = "freeworld"

var signingTime = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func TestSignedSeedVerifiesTheIdentityOfItsKey(t *testing.T) {
	key := peerkeytest.Key(0)

	identity, verified := signedSeed(t, key).VerifiedIdentityIn(signedNetworkName)
	if !verified {
		t.Fatal("VerifiedIdentityIn = unverified, want the identity of the signing key")
	}
	if identity != key.Identity() {
		t.Errorf("VerifiedIdentityIn = %v, want %v", identity, key.Identity())
	}
}

func TestSignedSeedKeepsItsIdentityWhenJavaMovesItsHosts(t *testing.T) {
	seed := signedSeed(t, peerkeytest.Key(0))
	seed.PrimaryHost = yacymodel.Some(hostOf(t, "2001:db8::1"))
	seed.AdditionalHosts = yacymodel.Some([]yacymodel.Host{
		hostOf(t, "PEER.example"),
		hostOf(t, "::ffff:192.0.2.1"),
	})

	if _, verified := seed.VerifiedIdentityIn(signedNetworkName); !verified {
		t.Error("VerifiedIdentityIn = unverified, want the hosts read as one set")
	}
}

func TestSignedSeedLosesItsIdentityWhenItsAddressChanges(t *testing.T) {
	for name, change := range map[string]func(*yacymodel.Seed){
		"host replaced": func(seed *yacymodel.Seed) {
			seed.PrimaryHost = yacymodel.Some(hostOf(t, "198.51.100.7"))
		},
		"host pruned": func(seed *yacymodel.Seed) {
			seed.AdditionalHosts = yacymodel.None[[]yacymodel.Host]()
		},
		"host added": func(seed *yacymodel.Seed) {
			seed.AdditionalHosts = yacymodel.Some([]yacymodel.Host{
				hostOf(t, "2001:db8::1"), hostOf(t, "peer.example"), hostOf(t, "198.51.100.7"),
			})
		},
		"port": func(seed *yacymodel.Seed) {
			seed.Port = yacymodel.Some(yacymodel.Port(8091))
		},
		"secure port": func(seed *yacymodel.Seed) {
			seed.SecurePort = yacymodel.Some(yacymodel.Port(8443))
		},
		"issue time": func(seed *yacymodel.Seed) {
			signature, _ := seed.AddressSignature.Get()
			signature.IssuedAt = signature.IssuedAt.Add(time.Second)
			seed.AddressSignature = yacymodel.Some(signature)
		},
		"hash": func(seed *yacymodel.Seed) {
			seed.Hash = peerkeytest.Key(1).Identity().Hash()
		},
		"identity": func(seed *yacymodel.Seed) {
			seed.Identity = yacymodel.Some(peerkeytest.Key(1).Identity())
		},
		"no identity": func(seed *yacymodel.Seed) {
			seed.Identity = yacymodel.None[yacymodel.PeerIdentity]()
		},
		"no signature": func(seed *yacymodel.Seed) {
			seed.AddressSignature = yacymodel.None[yacymodel.AddressSignature]()
		},
	} {
		t.Run(name, func(t *testing.T) {
			seed := signedSeed(t, peerkeytest.Key(0))
			change(&seed)

			if _, verified := seed.VerifiedIdentityIn(signedNetworkName); verified {
				t.Error("VerifiedIdentityIn = verified, want unverified")
			}
		})
	}
}

func TestSignedSeedLosesItsIdentityInAnotherNetwork(t *testing.T) {
	seed := signedSeed(t, peerkeytest.Key(0))

	if _, verified := seed.VerifiedIdentityIn("othernetwork"); verified {
		t.Error("VerifiedIdentityIn = verified in another network, want unverified")
	}
}

func TestAddressSignatureIssuesAtTheSecond(t *testing.T) {
	signature := peerkeytest.Key(0).AddressSignatureOf(
		unsignedSeedOf(t, peerkeytest.Key(0)),
		signedNetworkName,
		signingTime.Add(700*time.Millisecond),
	)

	if !signature.IssuedAt.Equal(signingTime) {
		t.Errorf("IssuedAt = %v, want %v", signature.IssuedAt, signingTime)
	}
}

func signedSeed(t *testing.T, key yacymodel.PeerKey) yacymodel.Seed {
	t.Helper()
	seed := unsignedSeedOf(t, key)
	seed.Identity = yacymodel.Some(key.Identity())
	seed.AddressSignature = yacymodel.Some(
		key.AddressSignatureOf(seed, signedNetworkName, signingTime),
	)

	return seed
}

func unsignedSeedOf(t *testing.T, key yacymodel.PeerKey) yacymodel.Seed {
	t.Helper()
	seed := addressableSeed(t)
	seed.Hash = key.Identity().Hash()
	seed.AdditionalHosts = yacymodel.Some([]yacymodel.Host{
		hostOf(t, "2001:db8::1"),
		hostOf(t, "peer.example"),
	})

	return seed
}
