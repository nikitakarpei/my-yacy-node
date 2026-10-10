// Package postingreplicas records which peers have accepted a copy of each
// stored posting.
package postingreplicas

import (
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
)

type Replicas struct {
	holders *vault.Collection[postingidentity.Identity, []yacymodel.Hash]
}

func Open(v *vault.Vault) (*Replicas, error) {
	holders, err := registerReplicaLedger(v)
	if err != nil {
		return nil, err
	}

	return &Replicas{holders: holders}, nil
}

func (l *Replicas) PostingPurged(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	identity := postingidentity.IdentityOf(posting)
	if _, err := l.holders.Delete(tx, identity); err != nil {
		return fmt.Errorf("drop replica ledger: %w", err)
	}

	return nil
}

func (l *Replicas) HoldersOf(
	tx *vault.Txn,
	identity postingidentity.Identity,
) ([]yacymodel.Hash, error) {
	holders, _, err := l.holders.Get(tx, identity)
	if err != nil {
		return nil, fmt.Errorf("read replica holders: %w", err)
	}

	return holders, nil
}

func (l *Replicas) RecordHolders(
	tx *vault.Txn,
	identity postingidentity.Identity,
	recordedHolders []yacymodel.Hash,
	holders []yacymodel.Hash,
) error {
	if slices.Equal(recordedHolders, holders) {
		return nil
	}
	if len(holders) == 0 {
		if _, err := l.holders.Delete(tx, identity); err != nil {
			return fmt.Errorf("drop replica holders: %w", err)
		}

		return nil
	}
	if _, err := l.holders.Put(tx, identity, holders); err != nil {
		return fmt.Errorf("record replica holders: %w", err)
	}

	return nil
}
