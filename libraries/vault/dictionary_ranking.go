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

const dictionaryRankingsBucket Name = "vault_fixed_dictionaries"

const fingerprintBytes = 8

var errBadDictionaryRanking = errors.New("bad dictionary ranking")

type dictionaryRanking struct {
	fingerprint   string
	encodedValues []byte
}

func (v *Vault) registerDictionaryRankings() (*Collection[Name, dictionaryRanking], error) {
	return v.RegisterCollection(dictionaryRankingsBucket, nameKeys, dictionaryRankingCodec{})
}

func (v *Vault) storedRankingOf(ctx context.Context, name Name) (dictionaryRanking, error) {
	rankings, err := v.dictionaryRankings()
	if err != nil {
		return dictionaryRanking{}, err
	}
	stored := rankingFrom(nil)
	if err := v.View(ctx, func(tx *Txn) error {
		ranking, found, err := rankings.Get(tx, name)
		if found {
			stored = ranking
		}

		return err
	}); err != nil {
		return dictionaryRanking{}, fmt.Errorf("read ranking of %s: %w", name, err)
	}

	return stored, nil
}

func rankingFrom(encodedValues []byte) dictionaryRanking {
	sum := sha256.Sum256(encodedValues)

	return dictionaryRanking{
		fingerprint:   hex.EncodeToString(sum[:fingerprintBytes]),
		encodedValues: encodedValues,
	}
}

func rankingOf[A any](values KeyPart[A], ranked []A) dictionaryRanking {
	var encodedValues []byte
	for _, value := range ranked {
		encodedValues = values.appendTo(encodedValues, value)
	}

	return rankingFrom(encodedValues)
}

func (v *Vault) keepRanking(ctx context.Context, name Name, ranking dictionaryRanking) error {
	rankings, err := v.dictionaryRankings()
	if err != nil {
		return err
	}
	if err := v.Update(ctx, func(tx *Txn) error {
		stored, found, err := rankings.Get(tx, name)
		if err != nil || (found && stored.fingerprint == ranking.fingerprint) {
			return err
		}
		if _, err := rankings.Put(tx, name, ranking); err != nil {
			return err
		}
		tx.RunAfterCommit(func() { reportRankingReplaced(ctx, name, stored, ranking) })

		return nil
	}); err != nil {
		return fmt.Errorf("keep ranking of %s: %w", name, err)
	}

	return nil
}

func reportRankingReplaced(ctx context.Context, name Name, previous, current dictionaryRanking) {
	slog.InfoContext(
		ctx,
		"vault fixed dictionary ranking replaced",
		slog.String("dictionary", string(name)),
		slog.String("previousFingerprint", previous.fingerprint),
		slog.String("fingerprint", current.fingerprint),
	)
}

type dictionaryRankingCodec struct{}

func (dictionaryRankingCodec) Encode(ranking dictionaryRanking) ([]byte, error) {
	encoded := binary.AppendUvarint(nil, uint64(len(ranking.fingerprint)))
	encoded = append(encoded, ranking.fingerprint...)

	return append(encoded, ranking.encodedValues...), nil
}

func (dictionaryRankingCodec) Decode(encoded []byte) (dictionaryRanking, error) {
	fingerprintLength, headerLength := binary.Uvarint(encoded)
	if headerLength <= 0 || fingerprintLength > uint64(len(encoded[headerLength:])) {
		return dictionaryRanking{}, errBadDictionaryRanking
	}
	stored := encoded[headerLength:]

	return dictionaryRanking{
		fingerprint:   string(stored[:fingerprintLength]),
		encodedValues: stored[fingerprintLength:],
	}, nil
}
