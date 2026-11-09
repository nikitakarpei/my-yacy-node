package companionprotocol_test

import (
	"encoding/json"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachPairingMessageKeepsItsWireForm(t *testing.T) {
	for _, message := range []struct {
		name     string
		value    any
		wireForm string
	}{
		{
			name: "pairing request",
			value: companionprotocol.PairingRequest{
				AppVersion:      "1.2",
				Attestation:     [][]byte{{1, 2}},
				Proof:           companionprotocol.Proof{3, 4},
				ProtocolVersion: 1,
			},
			wireForm: `{"appVersion":"1.2","attestation":["AQI="],"proof":"AwQ=","protocolVersion":1}`,
		},
		{
			name:     "accepted pairing reply",
			value:    companionprotocol.AcceptedPairingReply("device-a"),
			wireForm: `{"outcome":"accepted","deviceID":"device-a"}`,
		},
		{
			name:     "refused pairing reply",
			value:    companionprotocol.RefusedPairingReply(companionprotocol.ReasonAppOutdated),
			wireForm: `{"outcome":"refused","reason":"app-outdated"}`,
		},
	} {
		t.Run(message.name, func(t *testing.T) {
			encoded, err := json.Marshal(message.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			if string(encoded) != message.wireForm {
				t.Errorf("wire form = %s, want %s", encoded, message.wireForm)
			}
		})
	}
}
