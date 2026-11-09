package companionprotocol_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestPairingCredentialsSendTheDeviceNameAsTheClientNameAndTheSecretAsThePassword(t *testing.T) {
	credentials := companionprotocol.PairingCredentialsFrom("Pixel 9", "the-secret")

	if credentials.ClientName != "Pixel 9" || credentials.Password != "the-secret" {
		t.Errorf("client name = %q, password = %q, want Pixel 9 and the secret",
			credentials.ClientName, credentials.Password)
	}
}

func TestPairingCredentialsReadTheDeviceNameFromTheClientNameAndTheSecretFromThePassword(
	t *testing.T,
) {
	credentials := companionprotocol.PairingCredentials{
		ClientName: "Pixel 9",
		Password:   "the-secret",
	}

	if credentials.DeviceName() != "Pixel 9" || credentials.Secret() != "the-secret" {
		t.Errorf("credentials carry %q and %q, want Pixel 9 and the secret",
			credentials.DeviceName(), credentials.Secret())
	}
}
