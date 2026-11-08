package companionaccess_test

import (
	"maps"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
)

type brokerUsers map[string]bool

func (u brokerUsers) ConnectedUsers() map[string]bool { return u }

func TestDeviceConnectionsNameOnlyTheConnectedPairedDevices(t *testing.T) {
	users := brokerUsers{
		companionaccess.DeviceUserOf("device-a"): true,
		"pairing-secret":                         true,
		"":                                       true,
	}

	connected := companionaccess.NewDeviceConnections(users).ConnectedDevices()

	want := map[companionprotocol.DeviceID]bool{"device-a": true}
	if !maps.Equal(connected, want) {
		t.Errorf("connected devices = %v, want %v", connected, want)
	}
}
