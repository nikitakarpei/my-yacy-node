package companionaccess

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

type ConnectionLister interface {
	ConnectedUsers() map[string]bool
}

type DeviceConnections struct {
	broker ConnectionLister
}

func NewDeviceConnections(broker ConnectionLister) DeviceConnections {
	return DeviceConnections{broker: broker}
}

func (c DeviceConnections) ConnectedDevices() map[companionprotocol.DeviceID]bool {
	connectedDevices := map[companionprotocol.DeviceID]bool{}
	for user := range c.broker.ConnectedUsers() {
		if deviceID, isDevice := strings.CutPrefix(user, deviceUserStart); isDevice {
			connectedDevices[companionprotocol.DeviceID(deviceID)] = true
		}
	}
	return connectedDevices
}
