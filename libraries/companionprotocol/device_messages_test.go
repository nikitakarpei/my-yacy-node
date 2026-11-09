package companionprotocol_test

import (
	"encoding/json"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachDeviceMessageKeepsItsWireForm(t *testing.T) {
	for _, message := range []struct {
		name     string
		value    any
		wireForm string
	}{
		{
			name:     "hello",
			value:    companionprotocol.Hello{AppVersion: "1.2", ProtocolVersion: 1},
			wireForm: `{"appVersion":"1.2","protocolVersion":1}`,
		},
		{
			name:     "accepted hello reply",
			value:    companionprotocol.AcceptedHelloReply(),
			wireForm: `{"outcome":"accepted"}`,
		},
		{
			name:     "refused hello reply",
			value:    companionprotocol.RefusedHelloReply(companionprotocol.ReasonPeerOutdated),
			wireForm: `{"outcome":"refused","reason":"peer-outdated"}`,
		},
		{
			name:     "device settings",
			value:    companionprotocol.DeviceSettings{PartitionExponent: 6},
			wireForm: `{"partitionExponent":6}`,
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
