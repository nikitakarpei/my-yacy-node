package vault

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const lengthBucket = Name("__lengths__")

func lengthsOf(tx *Txn, collections []Name) (map[Name]int, error) {
	lengths := make(map[Name]int, len(collections))
	for _, bucket := range collections {
		length, err := readLength(tx, bucket)
		if err != nil {
			return nil, fmt.Errorf("length of %s: %w", bucket, err)
		}
		lengths[bucket] = length
	}

	return lengths, nil
}

func readLength(tx *Txn, bucket Name) (int, error) {
	return decodeLength(tx.etx.Bucket(lengthBucket).Get([]byte(bucket)))
}

func decodeLength(raw []byte) (int, error) {
	if raw == nil {
		return 0, nil
	}
	if len(raw) != 8 {
		return 0, fmt.Errorf("bad length counter: %d bytes", len(raw))
	}

	n := binary.BigEndian.Uint64(raw)
	if n > uint64(int(^uint(0)>>1)) {
		return 0, errors.New("length counter overflow")
	}

	return int(n), nil
}

func adjustLength(tx *Txn, bucket Name, delta int) error {
	lengths := tx.etx.Bucket(lengthBucket)
	current, err := decodeLength(lengths.Get([]byte(bucket)))
	if err != nil {
		return err
	}

	return putLength(lengths, []byte(bucket), max(current+delta, 0))
}

func putLength(lengths EngineBucket, key []byte, n int) error {
	if n < 0 {
		return errors.New("negative length counter")
	}

	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], uint64(n))
	if err := lengths.Put(key, raw[:]); err != nil {
		return fmt.Errorf("store length counter: %w", err)
	}

	return nil
}
