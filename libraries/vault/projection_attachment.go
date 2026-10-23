package vault

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

var (
	errProjectionAttachedTwice = errors.New("projection already attached to a collection")
	errProjectionUnattached    = errors.New("projection attached to no collection")
)

func (v *Vault) attachProjections[K, V any](
	ctx context.Context,
	collection Name,
	projections attachedProjections[K, V],
	storedRecordsIn func(*Txn) storedRecords,
) error {
	buckets := projections.buckets()
	if len(buckets) == 0 {
		return nil
	}
	if err := attachEach(buckets); err != nil {
		return err
	}

	return v.rebuildStaleProjections(ctx, collection, projections, storedRecordsIn)
}

func attachEach(buckets []projectionBucket) error {
	for _, bucket := range buckets {
		if err := bucket.attach(); err != nil {
			return err
		}
	}

	return nil
}

type projectionRegistration struct {
	bucket   Name
	version  string
	attached atomic.Bool
}

func (r *projectionRegistration) bucketName() Name {
	return r.bucket
}

func (r *projectionRegistration) attach() error {
	if r.attached.Swap(true) {
		return fmt.Errorf("%w: %s", errProjectionAttachedTwice, r.bucket)
	}

	return nil
}

func (r *projectionRegistration) registeredVersion() string {
	return r.version
}

func (r *projectionRegistration) refuseUnattached() error {
	if !r.attached.Load() {
		return fmt.Errorf("%w: %s", errProjectionUnattached, r.bucket)
	}

	return nil
}
