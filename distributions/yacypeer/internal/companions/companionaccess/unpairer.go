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

type Unpairer struct {
	devices RemovableDevices
	broker  Disconnector
}

func NewUnpairer(devices RemovableDevices, broker Disconnector) Unpairer {
	return Unpairer{devices: devices, broker: broker}
}

func (u Unpairer) Unpair(ctx context.Context, deviceID companionprotocol.DeviceID) {
	u.devices.Remove(ctx, deviceID)
	u.broker.DisconnectUser(DeviceUserOf(deviceID))
}
