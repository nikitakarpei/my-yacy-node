package attestedstatements

type HardwareKeys interface {
	CurrentKey() (HardwareKey, error)
}

type HardwareKey interface {
	KeyEvidence() []byte
	PublicKeyInfo() []byte
	Sign(message []byte) ([]byte, error)
}
