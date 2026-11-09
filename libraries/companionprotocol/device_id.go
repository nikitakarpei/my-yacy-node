package companionprotocol

type DeviceID string

func DeviceIDFrom(publicKeyInfo []byte) DeviceID {
	return DeviceID(shortHashFrom(publicKeyInfo))
}
