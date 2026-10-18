package vault

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

var (
	errDuplicateDictionary = errors.New("dictionary already registered")
	errRepeatedValue       = errors.New("value listed twice")
	errIDPastValues        = errors.New("id past the dictionary values")
)

type FixedDictionary[A any] struct {
	valuePart    KeyPart[A]
	valuesByID   []A
	idByEncoding map[string]uint64
	fingerprint  string
}

func (v *Vault) RegisterFixedDictionary[A any](
	ctx context.Context,
	name Name,
	valuePart KeyPart[A],
	valuesByID []A,
) (*FixedDictionary[A], error) {
	if err := v.claimDictionary(name); err != nil {
		return nil, err
	}
	keptCopy, err := v.keptCopyFor(ctx, name, valuePart, valuesByID)
	if err != nil {
		return nil, err
	}
	dictionary, err := fixedDictionaryFrom(valuePart, keptCopy)
	if err != nil {
		return nil, fmt.Errorf("dictionary %s: %w", name, err)
	}
	reportDictionaryRegistered(ctx, name, len(dictionary.valuesByID), keptCopy.fingerprint)

	return dictionary, nil
}

func (v *Vault) claimDictionary(name Name) error {
	if v == nil || v.engine == nil {
		return errVaultClosed
	}
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, claimed := v.claimedDictionaries[name]; claimed {
		return fmt.Errorf("%w: %s", errDuplicateDictionary, name)
	}
	v.claimedDictionaries[name] = struct{}{}

	return nil
}

func (v *Vault) keptCopyFor[A any](
	ctx context.Context,
	name Name,
	valuePart KeyPart[A],
	valuesByID []A,
) (dictionaryCopy, error) {
	if valuesByID == nil {
		return v.storedCopyOf(ctx, name)
	}
	givenCopy := copyFrom(valuePart, valuesByID)

	return givenCopy, v.keepCopy(ctx, name, givenCopy)
}

func fixedDictionaryFrom[A any](
	valuePart KeyPart[A],
	keptCopy dictionaryCopy,
) (*FixedDictionary[A], error) {
	dictionary := &FixedDictionary[A]{
		valuePart:    valuePart,
		idByEncoding: map[string]uint64{},
		fingerprint:  keptCopy.fingerprint,
	}
	unread := orderedBytes{encoded: keptCopy.encodedValues}
	for len(unread.encoded) > 0 {
		value, rest, err := valuePart.parseFrom(unread)
		if err != nil {
			return nil, fmt.Errorf("decode dictionary value: %w", err)
		}
		if err := dictionary.appendValue(
			value,
			unread.encoded[:len(unread.encoded)-len(rest.encoded)],
		); err != nil {
			return nil, err
		}
		unread = rest
	}

	return dictionary, nil
}

func (d *FixedDictionary[A]) appendValue(value A, encoding []byte) error {
	if _, listed := d.idByEncoding[string(encoding)]; listed {
		return fmt.Errorf("%w: at %d", errRepeatedValue, len(d.valuesByID))
	}
	d.idByEncoding[string(encoding)] = uint64(len(d.valuesByID))
	d.valuesByID = append(d.valuesByID, value)

	return nil
}

func reportDictionaryRegistered(
	ctx context.Context,
	name Name,
	valueAmount int,
	fingerprint string,
) {
	slog.InfoContext(
		ctx,
		"vault fixed dictionary registered",
		slog.String("dictionary", string(name)),
		slog.Int("values", valueAmount),
		slog.String("fingerprint", fingerprint),
	)
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
