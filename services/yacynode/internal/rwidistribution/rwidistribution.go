// Package rwidistribution opens the two durable records a distribution cycle
// needs for each stored posting — its offer schedule and its replica ledger —
// and fans out posting arrival and departure to both.
package rwidistribution

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
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

func (r *postingRecords) PostingStored(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	return r.schedule.PostingStored(tx, word, url)
}

func (r *postingRecords) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	if err := r.schedule.PostingPurged(tx, word, url); err != nil {
		return err
	}

	return r.replicas.PostingPurged(tx, word, url)
}
