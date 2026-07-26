package rwidistribution

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type replicaListCodec struct{}

func (replicaListCodec) Encode(replicas []yacymodel.Hash) ([]byte, error) {
	raw := make([]byte, 0, len(replicas)*yacymodel.HashLength)
	for _, replica := range replicas {
		raw = append(raw, replica.String()...)
	}

	return raw, nil
}

func (replicaListCodec) Decode(raw []byte) ([]yacymodel.Hash, error) {
	if len(raw)%yacymodel.HashLength != 0 {
		return nil, fmt.Errorf(
			"replica list: length %d not a multiple of %d",
			len(raw),
			yacymodel.HashLength,
		)
	}

	replicas := make([]yacymodel.Hash, 0, len(raw)/yacymodel.HashLength)
	for offset := 0; offset < len(raw); offset += yacymodel.HashLength {
		hash, err := yacymodel.ParseHash(string(raw[offset : offset+yacymodel.HashLength]))
		if err != nil {
			return nil, fmt.Errorf("replica list: %w", err)
		}
		replicas = append(replicas, hash)
	}

	return replicas, nil
}
