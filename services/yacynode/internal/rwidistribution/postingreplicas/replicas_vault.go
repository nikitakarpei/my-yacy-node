package postingreplicas

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
)

const bucket vault.Name = "rwidistribution_replica_ledger"

func registerReplicaLedger(
	v *vault.Vault,
) (*vault.Collection[postingidentity.Identity, []yacymodel.Hash], error) {
	holders, err := v.RegisterCollection(
		bucket,
		postingidentity.KeyLayout,
		holdersValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register replica ledger: %w", err)
	}

	return holders, nil
}

var errBadReplicaLedger = errors.New("bad replica ledger")

type holdersValueCodec struct{}

func (holdersValueCodec) Encode(holders []yacymodel.Hash) ([]byte, error) {
	var stored storedfields.Writer
	stored.Count(len(holders))
	for _, peer := range holders {
		stored.Fixed(peer.Bytes())
	}

	return stored.Record(), nil
}

func (holdersValueCodec) Decode(raw []byte) ([]yacymodel.Hash, error) {
	stored := storedfields.ReaderOf(raw, errBadReplicaLedger)
	holders := holdersFrom(stored)
	if err := stored.Err(); err != nil {
		return nil, err
	}

	return holders, nil
}

func holdersFrom(stored *storedfields.Reader) []yacymodel.Hash {
	count := stored.Count("holder count")
	holdersThatFit := stored.BytesLeft() / yacymodel.HashByteLength
	holders := make([]yacymodel.Hash, 0, min(count, holdersThatFit))
	for range count {
		holder, err := yacymodel.ParseHashBytes(stored.Fixed("holder", yacymodel.HashByteLength))
		if err != nil {
			stored.Reject("holder", err)

			return nil
		}
		holders = append(holders, holder)
	}

	return holders
}
