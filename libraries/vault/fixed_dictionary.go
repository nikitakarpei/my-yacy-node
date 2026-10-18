package vault

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

var (
	errDuplicateDictionary = errors.New("dictionary already registered")
	errRepeatedRankedValue = errors.New("value ranked twice")
	errUnrankedID          = errors.New("id past the ranked values")
)

type FixedDictionary[A any] struct {
	values       KeyPart[A]
	rankedValues []A
	idByEncoding map[string]uint64
	fingerprint  string
}

func (v *Vault) RegisterFixedDictionary[A any](
	ctx context.Context,
	name Name,
	values KeyPart[A],
	ranked []A,
) (*FixedDictionary[A], error) {
	if err := v.claimDictionary(name); err != nil {
		return nil, err
	}
	ranking, err := v.rankingFor(ctx, name, values, ranked)
	if err != nil {
		return nil, err
	}
	dictionary, err := fixedDictionaryFrom(values, ranking)
	if err != nil {
		return nil, fmt.Errorf("dictionary %s: %w", name, err)
	}
	reportDictionaryRegistered(ctx, name, len(dictionary.rankedValues), ranking.fingerprint)

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

func (v *Vault) rankingFor[A any](
	ctx context.Context,
	name Name,
	values KeyPart[A],
	ranked []A,
) (dictionaryRanking, error) {
	if ranked == nil {
		return v.storedRankingOf(ctx, name)
	}
	ranking := rankingOf(values, ranked)

	return ranking, v.keepRanking(ctx, name, ranking)
}

func fixedDictionaryFrom[A any](
	values KeyPart[A],
	ranking dictionaryRanking,
) (*FixedDictionary[A], error) {
	dictionary := &FixedDictionary[A]{
		values:       values,
		idByEncoding: map[string]uint64{},
		fingerprint:  ranking.fingerprint,
	}
	unread := orderedBytes{encoded: ranking.encodedValues}
	for len(unread.encoded) > 0 {
		value, rest, err := values.parseFrom(unread)
		if err != nil {
			return nil, fmt.Errorf("decode ranked value: %w", err)
		}
		if err := dictionary.rank(
			value,
			unread.encoded[:len(unread.encoded)-len(rest.encoded)],
		); err != nil {
			return nil, err
		}
		unread = rest
	}

	return dictionary, nil
}

func (d *FixedDictionary[A]) rank(value A, encoding []byte) error {
	if _, ranked := d.idByEncoding[string(encoding)]; ranked {
		return fmt.Errorf("%w: at %d", errRepeatedRankedValue, len(d.rankedValues))
	}
	d.idByEncoding[string(encoding)] = uint64(len(d.rankedValues))
	d.rankedValues = append(d.rankedValues, value)

	return nil
}

func reportDictionaryRegistered(ctx context.Context, name Name, values int, fingerprint string) {
	slog.InfoContext(
		ctx,
		"vault fixed dictionary registered",
		slog.String("dictionary", string(name)),
		slog.Int("values", values),
		slog.String("fingerprint", fingerprint),
	)
}

func (d *FixedDictionary[A]) storedMemberOf(value A) StoredMember {
	encoding := d.values.encodingOf(value)
	if id, ranked := d.idByEncoding[string(encoding)]; ranked {
		return StoredMember{ID: id}
	}

	return StoredMember{Raw: string(encoding)}
}

func (d *FixedDictionary[A]) valueOf(member StoredMember) (A, error) {
	if member.form() == MemberFormRaw {
		value, rest, err := d.values.parseFrom(orderedBytes{encoded: []byte(member.Raw)})
		if err != nil {
			return value, err
		}

		return value, rest.leftoverError()
	}
	if member.ID >= uint64(len(d.rankedValues)) {
		var unranked A

		return unranked, fmt.Errorf("%w: %d of %d", errUnrankedID, member.ID, len(d.rankedValues))
	}

	return d.rankedValues[member.ID], nil
}
