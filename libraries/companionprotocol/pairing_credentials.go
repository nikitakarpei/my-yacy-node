package companionprotocol

type PairingCredentials struct {
	ClientName string
	Password   string
}

func PairingCredentialsFrom(deviceName DeviceName, secret Secret) PairingCredentials {
	return PairingCredentials{ClientName: string(deviceName), Password: string(secret)}
}

func (c PairingCredentials) DeviceName() DeviceName {
	return DeviceName(c.ClientName)
}

func (c PairingCredentials) Secret() Secret {
	return Secret(c.Password)
}
