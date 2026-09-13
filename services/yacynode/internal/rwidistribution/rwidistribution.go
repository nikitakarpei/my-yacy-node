// Package rwidistribution opens the two durable records a distribution cycle
// needs for each stored posting — its offer schedule and its replica ledger —
// and tells both when a posting is stored and when a posting is purged. When a
// posting is updated, it becomes due for a new offer, and its replica ledger
// does not change.
package rwidistribution

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

func Open(v *vault.Vault, now func() time.Time, observer postingofferschedule.Observer) (
	*postingofferschedule.Schedule,
	*postingreplicas.Replicas,
	rwipostings.PostingObserver,
	error,
) {
	schedule, err := postingofferschedule.Open(v, now, observer)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open offer schedule: %w", err)
	}

	replicas, err := postingreplicas.Open(v, schedule)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open replica ledger: %w", err)
	}

	return schedule, replicas, &postingRecords{
		schedule: schedule,
		replicas: replicas,
	}, nil
}

type postingRecords struct {
	schedule *postingofferschedule.Schedule
	replicas *postingreplicas.Replicas
}

func (r *postingRecords) PostingStored(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	return r.schedule.PostingStored(tx, posting)
}

func (r *postingRecords) PostingUpdated(
	tx *vault.Txn,
	previous yacymodel.RWIPosting,
	current yacymodel.RWIPosting,
) error {
	return r.schedule.PostingUpdated(tx, previous, current)
}

func (r *postingRecords) PostingPurged(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if err := r.schedule.PostingPurged(tx, posting); err != nil {
		return err
	}

	return r.replicas.PostingPurged(tx, posting)
}
