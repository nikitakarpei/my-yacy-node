package boltvault

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	bolt "go.etcd.io/bbolt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	lengthBucket = vault.Name("__lengths__")
	lengthWidth  = 8
)

func lengthOf(lengths *bolt.Bucket, bucket vault.Name) (int, error) {
	return decodedLength(lengths.Get([]byte(bucket)))
}

func decodedLength(raw []byte) (int, error) {
	if raw == nil {
		return 0, nil
	}
	if len(raw) != lengthWidth {
		return 0, fmt.Errorf("bad length counter: %d bytes", len(raw))
	}

	stored := binary.BigEndian.Uint64(raw)
	if stored > uint64(math.MaxInt) {
		return 0, errors.New("length counter overflow")
	}

	return int(stored), nil
}

func adjustLength(lengths *bolt.Bucket, bucket vault.Name, delta int) error {
	current, err := lengthOf(lengths, bucket)
	if err != nil {
		return err
	}

	var raw [lengthWidth]byte
	binary.BigEndian.PutUint64(raw[:], uint64(max(current+delta, 0)))
	if err := lengths.Put([]byte(bucket), raw[:]); err != nil {
		return fmt.Errorf("store length counter: %w", err)
	}

	return nil
}
