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
	holders  *vault.Collection[[]yacymodel.Hash]
	schedule *postingschedule.Schedule
}

func Open(v *vault.Vault, schedule *postingschedule.Schedule) (*Replicas, error) {
	holders, err := vault.Register(v, bucket, holdersCodec{})
	if err != nil {
		return nil, fmt.Errorf("register replica ledger: %w", err)
	}

	return &Replicas{vault: v, holders: holders, schedule: schedule}, nil
}

func (l *Replicas) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	if _, err := l.holders.Delete(tx, postingschedule.PostingKey(word, url)); err != nil {
		return fmt.Errorf("drop replica ledger: %w", err)
	}

	return nil
}

// Holders reports the peers that have accepted a copy of the posting.
func (l *Replicas) Holders(
	ctx context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) ([]yacymodel.Hash, error) {
	var holders []yacymodel.Hash
	err := l.vault.View(ctx, func(tx *vault.Txn) error {
		stored, _, err := l.holders.Get(tx, postingschedule.PostingKey(word, url))
		holders = stored

		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read replica holders: %w", err)
	}

	return holders, nil
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
			holders, _, err := l.holders.Get(tx, key)
			if err != nil {
				return fmt.Errorf("read replica holders: %w", err)
			}
			if slices.Contains(holders, peer) {
				continue
			}
			if err := l.holders.Put(tx, key, append(holders, peer)); err != nil {
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
		holders, found, err := l.holders.Get(tx, key)
		if err != nil {
			return fmt.Errorf("read replica holders: %w", err)
		}
		if !found {
			return nil
		}

		kept := make([]yacymodel.Hash, 0, len(holders))
		for _, peer := range holders {
			if slices.Contains(stale, peer) {
				continue
			}
			kept = append(kept, peer)
		}
		dropped = len(holders) - len(kept)

		if dropped == 0 {
			return nil
		}
		if len(kept) == 0 {
			_, err := l.holders.Delete(tx, key)

			return err
		}

		return l.holders.Put(tx, key, kept)
	})
	if err != nil {
		return 0, fmt.Errorf("drop stale replicas: %w", err)
	}

	return dropped, nil
}
