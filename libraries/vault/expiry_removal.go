package vault

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

const expiredRecordsPerChunk = 1000

type expiryRemovals struct {
	guard   sync.Mutex
	stops   []context.CancelFunc
	running sync.WaitGroup
}

func (r *expiryRemovals) start(ctx context.Context, removal expiryRemoval) {
	removalContext, stop := context.WithCancel(context.WithoutCancel(ctx))

	r.guard.Lock()
	defer r.guard.Unlock()

	r.stops = append(r.stops, stop)
	r.running.Go(func() { removal.run(removalContext) })
}

func (r *expiryRemovals) stop() {
	r.guard.Lock()
	for _, stop := range r.stops {
		stop()
	}
	r.stops = nil
	r.guard.Unlock()

	r.running.Wait()
}

type expiringBucket interface {
	expiredKeysFrom(tx *Txn, resume []byte) (expiredKeys [][]byte, nextResume []byte, err error)
	forgetExpired(tx *Txn, expiredKeys [][]byte) (amountForgotten int, err error)
}

type expiryRemoval struct {
	storage  *Vault
	clock    Clock
	records  expiringBucket
	observer ExpiryObserver
	bucket   Name
	interval time.Duration
}

func (r expiryRemoval) run(ctx context.Context) {
	for {
		r.forgetExpiredRecords(ctx)
		if err := r.clock.Sleep(ctx, r.interval); err != nil {
			return
		}
	}
}

func (r expiryRemoval) forgetExpiredRecords(ctx context.Context) {
	r.observer.ExpiryPassBegan(ctx, r.bucket, r.clock.Now())

	var resume []byte
	for {
		expiredKeys, nextResume, err := r.expiredKeysFrom(ctx, resume)
		if err == nil {
			err = r.forgetDecodable(ctx, expiredKeys)
		}
		if err != nil {
			r.reportUnlessStopped(ctx, err)

			return
		}
		if nextResume == nil {
			break
		}
		resume = nextResume
	}

	r.observer.ExpiryPassFinished(ctx, r.bucket, r.clock.Now())
}

func (r expiryRemoval) expiredKeysFrom(
	ctx context.Context,
	resume []byte,
) (expiredKeys [][]byte, nextResume []byte, err error) {
	err = r.storage.View(ctx, func(tx *Txn) error {
		expiredKeys, nextResume, err = r.records.expiredKeysFrom(tx, resume)

		return err
	})

	return expiredKeys, nextResume, err
}

func (r expiryRemoval) forgetDecodable(ctx context.Context, expiredKeys [][]byte) error {
	for len(expiredKeys) > 0 {
		amountForgotten, err := r.forget(ctx, expiredKeys)
		if undecodable, isUndecodable := errors.AsType[undecodableRecordError](err); isUndecodable {
			amountListed := len(expiredKeys)
			expiredKeys = slices.DeleteFunc(expiredKeys, func(key []byte) bool {
				return bytes.Equal(key, undecodable.key)
			})
			if len(expiredKeys) == amountListed {
				return undecodable
			}
			r.observer.ExpiredRecordsNotForgotten(ctx, r.bucket, undecodable)

			continue
		}
		if err != nil {
			return err
		}
		if amountForgotten > 0 {
			r.observer.ExpiredRecordsForgotten(ctx, r.bucket, amountForgotten)
		}

		return nil
	}

	return nil
}

func (r expiryRemoval) forget(ctx context.Context, expiredKeys [][]byte) (int, error) {
	var amountForgotten int
	err := r.storage.Update(ctx, func(tx *Txn) error {
		var err error
		amountForgotten, err = r.records.forgetExpired(tx, expiredKeys)

		return err
	})

	return amountForgotten, err
}

func (r expiryRemoval) reportUnlessStopped(ctx context.Context, cause error) {
	if ctx.Err() != nil {
		return
	}
	r.observer.ExpiredRecordsNotForgotten(ctx, r.bucket, cause)
}
