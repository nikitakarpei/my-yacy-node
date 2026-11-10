package pageservices

import "github.com/nikitakarpei/yacy-rwi-node/attestedstatements"

type HardwareKeys interface {
	CurrentKey() (HardwareKey, error)
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
