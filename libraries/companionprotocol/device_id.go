package companionprotocol

type DeviceID string

func DeviceIDOf(publicKeyInfo []byte) DeviceID {
	return DeviceID(shortHashOf(publicKeyInfo))
}
