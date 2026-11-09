package companionprotocol_test

import (
	"encoding/json"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestAnAcceptedHelloReplyNamesNoReason(t *testing.T) {
	encoded, err := json.Marshal(companionprotocol.AcceptedHelloReply())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if want := `{"outcome":"accepted"}`; string(encoded) != want {
		t.Errorf("reply = %s, want %s", encoded, want)
	}
}

func TestARefusedHelloReplyNamesTheReason(t *testing.T) {
	encoded, err := json.Marshal(
		companionprotocol.RefusedHelloReply(companionprotocol.ReasonPeerOutdated),
	)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if want := `{"outcome":"refused","reason":"peer-outdated"}`; string(encoded) != want {
		t.Errorf("reply = %s, want %s", encoded, want)
	}
}
