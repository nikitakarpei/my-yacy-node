package vault

import (
	"bytes"
	"errors"
)

type Key struct {
	encoded []byte
}

func (key Key) Bytes() []byte {
	return bytes.Clone(key.encoded)
}

const keyCapacity = 48

var errKeyLayoutMismatch = errors.New("key does not match layout")
