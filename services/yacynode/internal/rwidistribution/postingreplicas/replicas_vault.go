package postingreplicas

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const bucket vault.Name = "rwidistribution_replica_ledger"

func registerReplicaLedger(
	v *vault.Vault,
) (*vault.Collection[postingidentity.Identity, []yacymodel.Hash], error) {
	holders, err := vault.Register(v, bucket, postingidentity.KeyCodec{}, holdersValueCodec{})
	if err != nil {
		return nil, fmt.Errorf("register replica ledger: %w", err)
	}

	return holders, nil
}

type holdersValueCodec struct{}

func (holdersValueCodec) Encode(holders []yacymodel.Hash) ([]byte, error) {
	raw := make([]byte, 0, len(holders)*yacymodel.HashLength)
	for _, peer := range holders {
		raw = append(raw, peer.String()...)
	}

	return raw, nil
}

func (holdersValueCodec) Decode(raw []byte) ([]yacymodel.Hash, error) {
	if len(raw)%yacymodel.HashLength != 0 {
		return nil, fmt.Errorf(
			"replica holders: length %d not a multiple of %d",
			len(raw),
			yacymodel.HashLength,
		)
	}

	holders := make([]yacymodel.Hash, 0, len(raw)/yacymodel.HashLength)
	for offset := 0; offset < len(raw); offset += yacymodel.HashLength {
		hash, err := yacymodel.ParseHash(string(raw[offset : offset+yacymodel.HashLength]))
		if err != nil {
			return nil, fmt.Errorf("replica holders: %w", err)
		}
		holders = append(holders, hash)
	}

	return holders, nil
}
