package pairingoffers

import "github.com/nikitakarpei/yacy-rwi-node/companionprotocol"

type Event interface {
	offerEvent()
}

type SecretRenewed struct {
	Secret companionprotocol.Secret
}

type PairingBegun struct {
	DeviceName companionprotocol.DeviceName
}

type DevicePaired struct {
	DeviceName companionprotocol.DeviceName
	Device     companionprotocol.DeviceID
}

type DeviceRefused struct {
	DeviceName companionprotocol.DeviceName
	Reason     companionprotocol.RefusalReason
}

func (SecretRenewed) offerEvent() {}

func (PairingBegun) offerEvent() {}

func (DevicePaired) offerEvent() {}

func (DeviceRefused) offerEvent() {}
