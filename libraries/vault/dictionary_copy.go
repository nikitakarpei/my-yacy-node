package vault

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
)

const dictionaryCopiesBucket Name = "vault_dictionary_copies"

const fingerprintBytes = 8

var errBadDictionaryCopy = errors.New("bad dictionary copy")

type dictionaryCopy struct {
	fingerprint   string
	encodedValues []byte
}

func (v *Vault) registerDictionaryCopies() (*Collection[Name, dictionaryCopy], error) {
	return v.RegisterCollection(dictionaryCopiesBucket, nameKeys, dictionaryCopyCodec{})
}

func (v *Vault) storedCopyOf(ctx context.Context, name Name) (dictionaryCopy, error) {
	copies, err := v.dictionaryCopies()
	if err != nil {
		return dictionaryCopy{}, err
	}
	var storedCopy dictionaryCopy
	var found bool
	if err := v.View(ctx, func(tx *Txn) error {
		var err error
		storedCopy, found, err = copies.Get(tx, name)

		return err
	}); err != nil {
		return dictionaryCopy{}, fmt.Errorf("read copy of %s: %w", name, err)
	}
	if !found {
		return fingerprinted(nil), nil
	}

	return storedCopy, nil
}

func fingerprinted(encodedValues []byte) dictionaryCopy {
	sum := sha256.Sum256(encodedValues)

	return dictionaryCopy{
		fingerprint:   hex.EncodeToString(sum[:fingerprintBytes]),
		encodedValues: encodedValues,
	}
}

func copyFrom[A any](valuePart KeyPart[A], valuesByID []A) dictionaryCopy {
	var encodedValues []byte
	for _, value := range valuesByID {
		encodedValues = valuePart.appendTo(encodedValues, value)
	}

	return fingerprinted(encodedValues)
}

func (v *Vault) keepCopy(ctx context.Context, name Name, givenCopy dictionaryCopy) error {
	copies, err := v.dictionaryCopies()
	if err != nil {
		return err
	}
	if err := v.Update(ctx, func(tx *Txn) error {
		storedCopy, found, err := copies.Get(tx, name)
		if err != nil || (found && storedCopy.fingerprint == givenCopy.fingerprint) {
			return err
		}
		if _, err := copies.Put(tx, name, givenCopy); err != nil {
			return err
		}
		tx.RunAfterCommit(func() { reportCopyReplaced(ctx, name, storedCopy, givenCopy) })

		return nil
	}); err != nil {
		return fmt.Errorf("keep copy of %s: %w", name, err)
	}

	return nil
}

func reportCopyReplaced(ctx context.Context, name Name, storedCopy, givenCopy dictionaryCopy) {
	slog.InfoContext(
		ctx,
		"vault fixed dictionary copy replaced",
		slog.String("dictionary", string(name)),
		slog.String("previousFingerprint", storedCopy.fingerprint),
		slog.String("fingerprint", givenCopy.fingerprint),
	)
}

type dictionaryCopyCodec struct{}

func (dictionaryCopyCodec) Encode(givenCopy dictionaryCopy) ([]byte, error) {
	encoded := binary.AppendUvarint(nil, uint64(len(givenCopy.fingerprint)))
	encoded = append(encoded, givenCopy.fingerprint...)

	return append(encoded, givenCopy.encodedValues...), nil
}

func (dictionaryCopyCodec) Decode(encoded []byte) (dictionaryCopy, error) {
	fingerprintLength, headerLength := binary.Uvarint(encoded)
	if headerLength <= 0 || fingerprintLength > uint64(len(encoded[headerLength:])) {
		return dictionaryCopy{}, errBadDictionaryCopy
	}
	fingerprintAndValues := encoded[headerLength:]

	return dictionaryCopy{
		fingerprint:   string(fingerprintAndValues[:fingerprintLength]),
		encodedValues: fingerprintAndValues[fingerprintLength:],
	}, nil
}
