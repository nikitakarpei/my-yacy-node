package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

const fingerprintBytes = 8

var (
	errRepeatedValue = errors.New("value listed twice")
	errIDPastValues  = errors.New("id past the dictionary values")
)

type FixedDictionary[A any] struct {
	valuePart    KeyPart[A]
	valuesByID   []A
	idByEncoding map[string]uint64
	fingerprint  string
}

func FixedDictionaryFrom[A any](valuePart KeyPart[A], valuesByID []A) (*FixedDictionary[A], error) {
	dictionary := &FixedDictionary[A]{
		valuePart:    valuePart,
		valuesByID:   valuesByID,
		idByEncoding: make(map[string]uint64, len(valuesByID)),
	}
	fingerprint := sha256.New()
	for id, value := range valuesByID {
		encoding := valuePart.encodingOf(value)
		if _, listed := dictionary.idByEncoding[string(encoding)]; listed {
			return nil, fmt.Errorf("%w: at %d", errRepeatedValue, id)
		}
		dictionary.idByEncoding[string(encoding)] = uint64(id)
		_, _ = fingerprint.Write(encoding)
	}
	dictionary.fingerprint = hex.EncodeToString(fingerprint.Sum(nil)[:fingerprintBytes])

	return dictionary, nil
}

func (d *FixedDictionary[A]) idOf(value A) (uint64, bool) {
	id, found := d.idByEncoding[string(d.encodingOf(value))]

	return id, found
}

func (d *FixedDictionary[A]) encodingOf(value A) []byte {
	return d.valuePart.encodingOf(value)
}

func (d *FixedDictionary[A]) valueAt(id uint64) (A, error) {
	if id >= uint64(len(d.valuesByID)) {
		var noValue A

		return noValue, fmt.Errorf("%w: %d of %d", errIDPastValues, id, len(d.valuesByID))
	}

	return d.valuesByID[id], nil
}

func (d *FixedDictionary[A]) valueFrom(encoding []byte) (A, error) {
	value, rest, err := d.valuePart.parseFrom(orderedBytes{encoded: encoding})
	if err != nil {
		return value, err
	}

	return value, rest.leftoverError()
}
