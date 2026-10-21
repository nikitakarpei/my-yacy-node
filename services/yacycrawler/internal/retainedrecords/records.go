package retainedrecords

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	expiryIndexVersion = "1"
	expiryIndexSuffix  = "_by_expiry"
)

var errRetainedRecordMalformed = errors.New("retained record has a malformed expiry")

var expiryKeyParts = vault.PairKey(vault.IntegerKeyPart, recordKeyPart)

var expiryEntries = expiryKeyParts.KeyLayoutFor(
	func(entry expiry) (int64, []byte) { return entry.keptUntil, entry.recordKey },
	func(keptUntil int64, recordKey []byte) expiry {
		return expiry{keptUntil: keptUntil, recordKey: recordKey}
	},
)

var recordKeyPart = vault.BytesKeyPartFrom(
	func(recordKey []byte, buffer []byte) []byte { return append(buffer, recordKey...) },
	func(stored vault.StoredBytes) ([]byte, error) { return stored.AppendTo(nil), nil },
)

type Records[K, V any] struct {
	name      vault.Name
	records   *vault.Collection[K, retained[V]]
	expiries  *vault.ValueIndex[expiry, expiry]
	keys      vault.KeyLayout[K]
	retention time.Duration
}

type retained[V any] struct {
	value     V
	keptUntil int64
}

type expiry struct {
	keptUntil int64
	recordKey []byte
}

func (r *Records[K, V]) Get(tx *vault.Txn, key K, now time.Time) (V, bool, error) {
	var absent V
	record, found, err := r.records.Get(tx, key)
	if err != nil {
		return absent, false, fmt.Errorf("read %s: %w", r.name, err)
	}
	if !found || record.keptUntil <= now.UnixMilli() {
		return absent, false, nil
	}
	return record.value, true, nil
}

func (r *Records[K, V]) Put(tx *vault.Txn, key K, value V, now time.Time) error {
	if _, err := r.records.Put(tx, key, retained[V]{
		value:     value,
		keptUntil: now.Add(r.retention).UnixMilli(),
	}); err != nil {
		return fmt.Errorf("write %s: %w", r.name, err)
	}
	return nil
}

func (r *Records[K, V]) bucket() string {
	return string(r.name)
}

func (r *Records[K, V]) forgetExpired(tx *vault.Txn, now time.Time, limit int) (int, error) {
	expiredKeys, err := r.expiredKeys(tx, now, limit)
	if err != nil {
		return 0, err
	}
	for _, key := range expiredKeys {
		if _, err := r.records.Delete(tx, key); err != nil {
			return 0, fmt.Errorf("forget expired %s: %w", r.name, err)
		}
	}
	return len(expiredKeys), nil
}

func (r *Records[K, V]) expiredKeys(tx *vault.Txn, now time.Time, limit int) ([]K, error) {
	var expiredKeys []K
	err := r.expiries.Scan(
		tx,
		expiryKeyParts.KeysThroughFirst(now.UnixMilli()),
		func(entry expiry) (bool, error) {
			key, err := r.keys.Decode(entry.recordKey)
			if err != nil {
				return false, fmt.Errorf("decode expired %s key: %w", r.name, err)
			}
			expiredKeys = append(expiredKeys, key)
			return len(expiredKeys) < limit, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("scan %s expiries: %w", r.name, err)
	}
	return expiredKeys, nil
}

type retainedCodec[V any] struct {
	values vault.ValueCodec[V]
}

func (c retainedCodec[V]) Encode(record retained[V]) ([]byte, error) {
	value, err := c.values.Encode(record.value)
	if err != nil {
		return nil, err
	}
	encoded := binary.AppendVarint(
		make([]byte, 0, binary.MaxVarintLen64+len(value)),
		record.keptUntil,
	)
	return append(encoded, value...), nil
}

func (c retainedCodec[V]) Decode(encoded []byte) (retained[V], error) {
	keptUntil, keptUntilBytes := binary.Varint(encoded)
	if keptUntilBytes <= 0 {
		return retained[V]{}, errRetainedRecordMalformed
	}
	value, err := c.values.Decode(encoded[keptUntilBytes:])
	if err != nil {
		return retained[V]{}, err
	}
	return retained[V]{value: value, keptUntil: keptUntil}, nil
}
