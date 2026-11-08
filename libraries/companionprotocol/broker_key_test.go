package companionprotocol_test

import (
	"bytes"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachKeyGivesItsOwnBrokerKeyHash(t *testing.T) {
	publicKeyInfo := randomPublicKeyInfo(t)

	if companionprotocol.BrokerKeyHashFrom(
		publicKeyInfo,
	) != companionprotocol.BrokerKeyHashFrom(
		bytes.Clone(publicKeyInfo),
	) {
		t.Error("one key gives two hashes")
	}
	if companionprotocol.BrokerKeyHashFrom(
		publicKeyInfo,
	) == companionprotocol.BrokerKeyHashFrom(
		randomPublicKeyInfo(t),
	) {
		t.Error("two keys give the same hash")
	}
}
