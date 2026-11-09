package companionprotocol_test

import (
	"encoding/json"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestAnAcceptedPairingReplyNamesTheDeviceAndNoReason(t *testing.T) {
	encoded, err := json.Marshal(companionprotocol.AcceptedPairingReply("device-a"))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if want := `{"outcome":"accepted","deviceID":"device-a"}`; string(encoded) != want {
		t.Errorf("reply = %s, want %s", encoded, want)
	}
}

func TestARefusedPairingReplyNamesTheReasonAndNoDevice(t *testing.T) {
	encoded, err := json.Marshal(
		companionprotocol.RefusedPairingReply(companionprotocol.ReasonAppOutdated),
	)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if want := `{"outcome":"refused","reason":"app-outdated"}`; string(encoded) != want {
		t.Errorf("reply = %s, want %s", encoded, want)
	}
}
