package companionprotocol_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestPairingCredentialsCarryTheDeviceNameAndTheSecret(t *testing.T) {
	secret := companionprotocol.RandomSecret()
	credentials := companionprotocol.PairingCredentialsFrom("Pixel 9", secret)

	read := companionprotocol.PairingCredentials{
		ClientName: credentials.ClientName,
		Password:   credentials.Password,
	}

	if read.DeviceName() != "Pixel 9" || read.Secret() != secret {
		t.Errorf("credentials carry %q and %q, want Pixel 9 and the secret",
			read.DeviceName(), read.Secret())
	}
}
