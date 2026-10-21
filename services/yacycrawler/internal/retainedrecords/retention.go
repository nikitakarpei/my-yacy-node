// Package retainedrecords keeps crawl state records in a vault for a retention
// renewed on every write. A record counts as absent once its retention ends,
// whether or not it is gone yet. A sweep forgets expired records in bounded
// batches, one transaction per batch, and reports what it forgot.
package retainedrecords

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	sweepInterval        = time.Minute
	expiredRecordsPerRun = 1000
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type ExpiryObserver interface {
	ExpiredRecordsForgotten(ctx context.Context, bucket string, amount int)
	ExpiredRecordsNotForgotten(ctx context.Context, bucket string, cause error)
}

type expiringRecords interface {
	bucket() string
	forgetExpired(tx *vault.Txn, now time.Time, limit int) (int, error)
}

type Retention struct {
	storage  *vault.Vault
	clock    Clock
	observer ExpiryObserver
	guard    sync.Mutex
	expiring []expiringRecords
}

func New(storage *vault.Vault, clock Clock, observer ExpiryObserver) *Retention {
	return &Retention{storage: storage, clock: clock, observer: observer}
}

func (r *Retention) Register[K, V any](
	ctx context.Context,
	bucket vault.Name,
	keys vault.KeyLayout[K],
	values vault.ValueCodec[V],
	keptFor time.Duration,
) (*Records[K, V], error) {
	expiries, err := r.storage.RegisterValueIndex(
		bucket+expiryIndexSuffix,
		expiryIndexVersion,
		expiryEntries,
		func(entry expiry) expiry { return entry },
	)
	if err != nil {
		return nil, fmt.Errorf("register %s expiries: %w", bucket, err)
	}
	kept, err := r.storage.RegisterProjectedCollection(
		ctx,
		bucket,
		keys,
		retainedCodec[V]{values: values},
		vault.ProjectedAs(func(key K, record retained[V]) expiry {
			return expiry{keptUntil: record.keptUntil, recordKey: keys.Encode(key).Bytes()}
		}, expiries),
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", bucket, err)
	}
	records := &Records[K, V]{
		name:      bucket,
		records:   kept,
		expiries:  expiries,
		keys:      keys,
		retention: keptFor,
	}
	r.guard.Lock()
	defer r.guard.Unlock()
	r.expiring = append(r.expiring, records)
	return records, nil
}

func (r *Retention) Run(ctx context.Context) error {
	for {
		r.forgetExpiredRecords(ctx)
		if err := r.clock.Sleep(ctx, sweepInterval); err != nil {
			return nil //nolint:nilerr // an ended context ends the sweep cleanly
		}
	}
}

func (r *Retention) forgetExpiredRecords(ctx context.Context) {
	now := r.clock.Now()
	for _, records := range r.registeredRecords() {
		r.forgetExpiredOf(ctx, records, now)
	}
}

func (r *Retention) registeredRecords() []expiringRecords {
	r.guard.Lock()
	defer r.guard.Unlock()
	return append([]expiringRecords(nil), r.expiring...)
}

func (r *Retention) forgetExpiredOf(ctx context.Context, records expiringRecords, now time.Time) {
	forgotten := 0
	for {
		amount, err := r.forgetExpiredBatch(ctx, records, now)
		forgotten += amount
		if err != nil {
			r.observer.ExpiredRecordsNotForgotten(ctx, records.bucket(), err)
			break
		}
		if amount < expiredRecordsPerRun {
			break
		}
	}
	if forgotten > 0 {
		r.observer.ExpiredRecordsForgotten(ctx, records.bucket(), forgotten)
	}
}

func (r *Retention) forgetExpiredBatch(
	ctx context.Context,
	records expiringRecords,
	now time.Time,
) (int, error) {
	var amount int
	err := r.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		amount, err = records.forgetExpired(tx, now, expiredRecordsPerRun)
		return err
	})
	if err != nil {
		return 0, err
	}
	return amount, nil
}
