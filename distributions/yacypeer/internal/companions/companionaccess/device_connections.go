package companionaccess

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

type ConnectedUsers interface {
	ConnectedUsers() map[string]bool
}

type DeviceConnections struct {
	users ConnectedUsers
}

func NewDeviceConnections(users ConnectedUsers) DeviceConnections {
	return DeviceConnections{users: users}
}

func (c DeviceConnections) ConnectedDevices() map[companionprotocol.DeviceID]bool {
	devices := map[companionprotocol.DeviceID]bool{}
	for user, connected := range c.users.ConnectedUsers() {
		if device, isDevice := strings.CutPrefix(user, deviceUserStart); isDevice && connected {
			devices[companionprotocol.DeviceID(device)] = true
		}
	}
	return devices
}
