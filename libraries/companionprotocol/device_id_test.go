package companionprotocol_test

import (
	"bytes"
	"crypto/x509"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachKeyGivesItsOwnDeviceID(t *testing.T) {
	publicKeyInfo := randomPublicKeyInfo(t)

	if companionprotocol.DeviceIDFrom(
		publicKeyInfo,
	) != companionprotocol.DeviceIDFrom(
		bytes.Clone(publicKeyInfo),
	) {
		t.Error("one key gives two device ids")
	}
	if companionprotocol.DeviceIDFrom(
		publicKeyInfo,
	) == companionprotocol.DeviceIDFrom(
		randomPublicKeyInfo(t),
	) {
		t.Error("two keys give the same device id")
	}
}

func randomPublicKeyInfo(t *testing.T) []byte {
	t.Helper()
	publicKeyInfo, err := x509.MarshalPKIXPublicKey(&randomDeviceKey(t).PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	return publicKeyInfo
}
