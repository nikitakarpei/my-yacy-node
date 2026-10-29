package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
)

var (
	endorsedTLSKey = []byte("DER public key of the endorsed TLS key")
	otherTLSKey    = []byte("DER public key of another TLS key")
)

func TestEndorsementOfATLSKeyEndorsesThatKeyInItsNetwork(t *testing.T) {
	endorsement := peerkeytest.Key(0).EndorsementOf(endorsedTLSKey, signedNetworkName)

	if endorsement.Identity != peerkeytest.Key(0).Identity() {
		t.Errorf("Identity = %v, want the identity of the endorsing key", endorsement.Identity)
	}
	if !endorsement.Endorses(endorsedTLSKey, signedNetworkName) {
		t.Error("Endorses = false, want true for the endorsed key")
	}
}

func TestEndorsementOfATLSKeyEndorsesNothingElse(t *testing.T) {
	endorsement := peerkeytest.Key(0).EndorsementOf(endorsedTLSKey, signedNetworkName)
	borrowed := endorsement
	borrowed.Identity = peerkeytest.Key(1).Identity()

	for name, check := range map[string]func() bool{
		"other TLS key": func() bool { return endorsement.Endorses(otherTLSKey, signedNetworkName) },
		"other network": func() bool { return endorsement.Endorses(endorsedTLSKey, "othernet") },
		"other identity": func() bool {
			return borrowed.Endorses(endorsedTLSKey, signedNetworkName)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if check() {
				t.Error("Endorses = true, want false")
			}
		})
	}
}

func TestEndorsementOfATLSKeyIsNotAnAddressSignature(t *testing.T) {
	seed := signedSeed(t, peerkeytest.Key(0))
	signature, _ := seed.AddressSignature.Get()
	endorsement := yacymodel.TLSKeyEndorsement{
		Identity:  peerkeytest.Key(0).Identity(),
		Signature: signature.Signature,
	}

	if endorsement.Endorses(endorsedTLSKey, signedNetworkName) {
		t.Error("Endorses = true for an address signature, want false")
	}
}
