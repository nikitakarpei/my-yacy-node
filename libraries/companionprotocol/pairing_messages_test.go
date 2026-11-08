package companionprotocol_test

import (
	"encoding/json"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestAnAcceptedReplyNamesTheDeviceAndNoReason(t *testing.T) {
	encoded, err := json.Marshal(companionprotocol.AcceptedReply("device-a"))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if want := `{"outcome":"accepted","device":"device-a"}`; string(encoded) != want {
		t.Errorf("reply = %s, want %s", encoded, want)
	}
}

func TestARefusedReplyNamesTheReasonAndNoDevice(t *testing.T) {
	encoded, err := json.Marshal(
		companionprotocol.RefusedReply(companionprotocol.ReasonAppOutdated),
	)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if want := `{"outcome":"refused","reason":"app-outdated"}`; string(encoded) != want {
		t.Errorf("reply = %s, want %s", encoded, want)
	}
}
