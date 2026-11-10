package statementsigning

type HardwareKeys interface {
	CurrentKey() (HardwareKey, error)
	TimeStampOf(signedStatementID []byte) ([]byte, error)
}

type HardwareKey interface {
	KeyEvidence() []byte
	PublicKeyInfo() []byte
	Sign(message []byte) ([]byte, error)
}
