// Package postingreplicas records which peers have accepted a copy of each
// stored posting.
package postingreplicas

import (
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const bucket vault.Name = "rwidistribution_replica_ledger"

type Replicas struct {
	holders  *vault.Collection[postingidentity.Identity, []yacymodel.Hash]
	schedule *postingofferschedule.Schedule
}

func Open(v *vault.Vault, schedule *postingofferschedule.Schedule) (*Replicas, error) {
	holders, err := vault.Register(v, bucket, postingidentity.KeyCodec{}, holdersValueCodec{})
	if err != nil {
		return nil, fmt.Errorf("register replica ledger: %w", err)
	}

	return &Replicas{holders: holders, schedule: schedule}, nil
}

func (l *Replicas) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	posting := postingidentity.IdentityOf(word, url)
	if _, err := l.holders.Delete(tx, posting); err != nil {
		return fmt.Errorf("drop replica ledger: %w", err)
	}

	return nil
}

func (l *Replicas) HoldersOf(
	tx *vault.Txn,
	posting postingidentity.Identity,
) ([]yacymodel.Hash, error) {
	holders, _, err := l.holders.Get(tx, posting)
	if err != nil {
		return nil, fmt.Errorf("read replica holders: %w", err)
	}

	return holders, nil
}

func (l *Replicas) RecordAccepted(
	tx *vault.Txn,
	peer yacymodel.Hash,
	postings []yacymodel.RWIPosting,
) error {
	for _, posting := range postings {
		identity := postingidentity.IdentityOf(posting.WordHash, posting.URLHash)
		postingScheduled, err := l.schedule.IsScheduled(tx, identity)
		if err != nil {
			return err
		}
		if !postingScheduled {
			continue
		}

		holders, _, err := l.holders.Get(tx, identity)
		if err != nil {
			return fmt.Errorf("read replica holders: %w", err)
		}
		if slices.Contains(holders, peer) {
			continue
		}
		if err := l.holders.Put(tx, identity, append(holders, peer)); err != nil {
			return fmt.Errorf("record accepted replica: %w", err)
		}
	}

	return nil
}

func (l *Replicas) DropStaleHolders(
	tx *vault.Txn,
	staleHolders map[postingidentity.Identity][]yacymodel.Hash,
) (int, error) {
	var droppedReplicas int
	for posting, peers := range staleHolders {
		droppedForPosting, err := l.dropHolders(tx, posting, peers)
		if err != nil {
			return 0, err
		}
		droppedReplicas += droppedForPosting
	}

	return droppedReplicas, nil
}

func (l *Replicas) dropHolders(
	tx *vault.Txn,
	posting postingidentity.Identity,
	staleHolders []yacymodel.Hash,
) (int, error) {
	holders, found, err := l.holders.Get(tx, posting)
	if err != nil {
		return 0, fmt.Errorf("read replica holders: %w", err)
	}
	if !found {
		return 0, nil
	}

	keptHolders := make([]yacymodel.Hash, 0, len(holders))
	for _, peer := range holders {
		if slices.Contains(staleHolders, peer) {
			continue
		}
		keptHolders = append(keptHolders, peer)
	}
	droppedReplicas := len(holders) - len(keptHolders)

	if droppedReplicas == 0 {
		return 0, nil
	}
	if len(keptHolders) == 0 {
		if _, err := l.holders.Delete(tx, posting); err != nil {
			return 0, fmt.Errorf("drop stale replicas: %w", err)
		}

		return droppedReplicas, nil
	}
	if err := l.holders.Put(tx, posting, keptHolders); err != nil {
		return 0, fmt.Errorf("drop stale replicas: %w", err)
	}

	return droppedReplicas, nil
}
