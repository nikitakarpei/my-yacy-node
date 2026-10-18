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

func (d *FixedDictionary[A]) storedMemberOf(value A) StoredMember {
	encoding := d.valuePart.encodingOf(value)
	if id, found := d.idByEncoding[string(encoding)]; found {
		return StoredMember{ID: id}
	}

	return StoredMember{Raw: string(encoding)}
}

func (d *FixedDictionary[A]) valueOf(member StoredMember) (A, error) {
	if member.form() == MemberFormRaw {
		value, rest, err := d.valuePart.parseFrom(orderedBytes{encoded: []byte(member.Raw)})
		if err != nil {
			return value, err
		}

		return value, rest.leftoverError()
	}
	if member.ID >= uint64(len(d.valuesByID)) {
		var noValue A

		return noValue, fmt.Errorf("%w: %d of %d", errIDPastValues, member.ID, len(d.valuesByID))
	}

	return d.valuesByID[member.ID], nil
}
