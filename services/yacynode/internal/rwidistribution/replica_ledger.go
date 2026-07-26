package rwidistribution

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const replicaLedgerBucket vault.Name = "rwidistribution_replica_ledger"

type replicaLedger struct {
	vault    *vault.Vault
	replicas *vault.Collection[[]yacymodel.Hash]
}

func openReplicaLedger(v *vault.Vault) (*replicaLedger, error) {
	replicas, err := vault.Register(v, replicaLedgerBucket, replicaListCodec{})
	if err != nil {
		return nil, fmt.Errorf("register replica ledger: %w", err)
	}

	return &replicaLedger{vault: v, replicas: replicas}, nil
}

func (l *replicaLedger) PostingStored(*vault.Txn, yacymodel.Hash, yacymodel.Hash) error {
	return nil
}

func (l *replicaLedger) PostingPurged(tx *vault.Txn, word, url yacymodel.Hash) error {
	if _, err := l.replicas.Delete(tx, postingKey(word, url)); err != nil {
		return fmt.Errorf("drop replica ledger: %w", err)
	}

	return nil
}

func (l *replicaLedger) Replicas(
	ctx context.Context,
	word, url yacymodel.Hash,
) ([]yacymodel.Hash, error) {
	var replicas []yacymodel.Hash
	err := l.vault.View(ctx, func(tx *vault.Txn) error {
		stored, _, err := l.replicas.Get(tx, postingKey(word, url))
		replicas = stored

		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read replicas: %w", err)
	}

	return replicas, nil
}

func (l *replicaLedger) RecordAccepted(ctx context.Context, word, url, peer yacymodel.Hash) error {
	err := l.vault.Update(ctx, func(tx *vault.Txn) error {
		key := postingKey(word, url)
		replicas, _, err := l.replicas.Get(tx, key)
		if err != nil {
			return fmt.Errorf("read replicas: %w", err)
		}
		for _, existing := range replicas {
			if existing == peer {
				return nil
			}
		}

		return l.replicas.Put(tx, key, append(replicas, peer))
	})
	if err != nil {
		return fmt.Errorf("record accepted replica: %w", err)
	}

	return nil
}

func (l *replicaLedger) Prune(
	ctx context.Context,
	word, url yacymodel.Hash,
	live func(peer yacymodel.Hash) bool,
) ([]yacymodel.Hash, error) {
	var remaining []yacymodel.Hash
	err := l.vault.Update(ctx, func(tx *vault.Txn) error {
		key := postingKey(word, url)
		replicas, found, err := l.replicas.Get(tx, key)
		if err != nil {
			return fmt.Errorf("read replicas: %w", err)
		}
		if !found {
			return nil
		}

		kept := make([]yacymodel.Hash, 0, len(replicas))
		for _, peer := range replicas {
			if live(peer) {
				kept = append(kept, peer)
			}
		}
		remaining = kept

		if len(kept) == len(replicas) {
			return nil
		}
		if len(kept) == 0 {
			_, err := l.replicas.Delete(tx, key)

			return err
		}

		return l.replicas.Put(tx, key, kept)
	})
	if err != nil {
		return nil, fmt.Errorf("prune replicas: %w", err)
	}

	return remaining, nil
}

var _ rwipostings.PostingObserver = (*replicaLedger)(nil)
