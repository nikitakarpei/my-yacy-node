package companionaccess_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
)

type unpairedRecords struct {
	removed      []companionprotocol.DeviceID
	disconnected []string
}

func (r *unpairedRecords) Remove(_ context.Context, id companionprotocol.DeviceID) {
	r.removed = append(r.removed, id)
}

func (r *unpairedRecords) DisconnectUser(user string) {
	r.disconnected = append(r.disconnected, user)
}

func TestUnpairingForgetsTheDeviceAndEndsItsConnections(t *testing.T) {
	records := &unpairedRecords{}

	companionaccess.NewUnpairer(records, records).Unpair(context.Background(), "device-a")

	if len(records.removed) != 1 || records.removed[0] != "device-a" {
		t.Errorf("removed = %v, want device-a", records.removed)
	}
	if len(records.disconnected) != 1 ||
		records.disconnected[0] != companionaccess.DeviceUserOf("device-a") {
		t.Errorf("disconnected = %v, want the user of device-a", records.disconnected)
	}
}
