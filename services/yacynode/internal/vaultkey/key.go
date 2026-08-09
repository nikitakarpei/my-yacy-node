// Package vaultkey encodes a collection key so that its byte order equals its domain order.
package vaultkey

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

var errUnorderableItem = errors.New("key layout offered an item that has no ordered encoding")

func keyOf(items []any) Key {
	encoded, err := orderedcode.Append(nil, items...)
	if err != nil {
		panic(fmt.Errorf("%w: %w", errUnorderableItem, err))
	}

	return Key{encoded: encoded}
}

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
