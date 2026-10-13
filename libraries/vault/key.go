package vault

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/google/orderedcode"
)

type Key struct {
	encoded []byte
}

func (key Key) Bytes() []byte {
	return bytes.Clone(key.encoded)
}

const keyCapacity = 48

var errKeyLayoutMismatch = errors.New("key does not match layout")

func parseInto(storedKey []byte, targets []any) error {
	remainder, err := orderedcode.Parse(string(storedKey), targets...)
	if err != nil {
		return fmt.Errorf("%w: %w", errKeyLayoutMismatch, err)
	}

	if remainder != "" {
		return fmt.Errorf("%w: %d trailing bytes", errKeyLayoutMismatch, len(remainder))
	}

	return nil
}
