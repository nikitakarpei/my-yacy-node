// Package postingreplicas records which peers have accepted a copy of each
// stored posting.
package postingreplicas

import (
	"context"
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const bucket vault.Name = "rwidistribution_replica_ledger"

type Replicas struct {
	vault    *vault.Vault
	replicas *vault.Collection[[]yacymodel.Hash]
	schedule *postingschedule.Schedule
}

func Open(v *vault.Vault, schedule *postingschedule.Schedule) (*Replicas, error) {
	replicas, err := vault.Register(v, bucket, replicaListCodec{})
	if err != nil {
		return nil, fmt.Errorf("register replica ledger: %w", err)
	}

	return &Replicas{vault: v, replicas: replicas, schedule: schedule}, nil
}

func (l *Replicas) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	if _, err := l.replicas.Delete(tx, postingschedule.PostingKey(word, url)); err != nil {
		return fmt.Errorf("drop replica ledger: %w", err)
	}

	return nil
}

func (l *Replicas) Replicas(
	ctx context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) ([]yacymodel.Hash, error) {
	var replicas []yacymodel.Hash
	err := l.vault.View(ctx, func(tx *vault.Txn) error {
		stored, _, err := l.replicas.Get(tx, postingschedule.PostingKey(word, url))
		replicas = stored

		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read replicas: %w", err)
	}

	return replicas, nil
}

func (l *Replicas) RecordAccepted(
	ctx context.Context,
	peer yacymodel.Hash,
	postings []yacymodel.RWIPosting,
) error {
	err := l.vault.Update(ctx, func(tx *vault.Txn) error {
		for _, posting := range postings {
			scheduled, err := l.schedule.Scheduled(tx, posting.WordHash, posting.URLHash)
			if err != nil {
				return err
			}
			if !scheduled {
				continue
			}

			key := postingschedule.PostingKey(posting.WordHash, posting.URLHash)
			replicas, _, err := l.replicas.Get(tx, key)
			if err != nil {
				return fmt.Errorf("read replicas: %w", err)
			}
			if slices.Contains(replicas, peer) {
				continue
			}
			if err := l.replicas.Put(tx, key, append(replicas, peer)); err != nil {
				return fmt.Errorf("record accepted replica: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("record accepted replicas: %w", err)
	}

	return nil
}

func (l *Replicas) RecordDropped(
	ctx context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	stale []yacymodel.Hash,
) (int, error) {
	var dropped int
	err := l.vault.Update(ctx, func(tx *vault.Txn) error {
		key := postingschedule.PostingKey(word, url)
		replicas, found, err := l.replicas.Get(tx, key)
		if err != nil {
			return fmt.Errorf("read replicas: %w", err)
		}
		if !found {
			return nil
		}

		kept := make([]yacymodel.Hash, 0, len(replicas))
		for _, peer := range replicas {
			if slices.Contains(stale, peer) {
				continue
			}
			kept = append(kept, peer)
		}
		dropped = len(replicas) - len(kept)

		if dropped == 0 {
			return nil
		}
		if len(kept) == 0 {
			_, err := l.replicas.Delete(tx, key)

			return err
		}

		return l.replicas.Put(tx, key, kept)
	})
	if err != nil {
		return 0, fmt.Errorf("drop stale replicas: %w", err)
	}

	return dropped, nil
}
