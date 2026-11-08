package companionaccess

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

type RemovableDevices interface {
	Remove(ctx context.Context, id companionprotocol.DeviceID)
}

type Disconnector interface {
	DisconnectUser(user string)
}

type Unpairing struct {
	devices RemovableDevices
	broker  Disconnector
}

func NewUnpairing(devices RemovableDevices, broker Disconnector) Unpairing {
	return Unpairing{devices: devices, broker: broker}
}

func (u Unpairing) Unpair(ctx context.Context, device companionprotocol.DeviceID) {
	u.devices.Remove(ctx, device)
	u.broker.DisconnectUser(UserOf(device))
}
