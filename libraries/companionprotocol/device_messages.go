package companionprotocol

type DeviceSettings struct {
	PartitionExponent uint `json:"partitionExponent"`
}

type AppReport struct {
	AppVersion string `json:"appVersion"`
}
