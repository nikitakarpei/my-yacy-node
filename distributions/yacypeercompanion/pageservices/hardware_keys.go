package pageservices

import "github.com/nikitakarpei/yacy-rwi-node/attestedstatements"

type HardwareKeys interface {
	CurrentKey() (HardwareKey, error)
	TimeStampOf(signedStatementID []byte) ([]byte, error)
}

type HardwareKey interface {
	KeyEvidence() []byte
	PublicKeyInfo() []byte
	Sign(message []byte) ([]byte, error)
}

type statementKeys struct {
	hardwareKeys HardwareKeys
}

func (k statementKeys) CurrentKey() (attestedstatements.HardwareKey, error) {
	return k.hardwareKeys.CurrentKey()
}

func (k statementKeys) TimeStampOf(signedStatementID []byte) ([]byte, error) {
	return k.hardwareKeys.TimeStampOf(signedStatementID)
}
