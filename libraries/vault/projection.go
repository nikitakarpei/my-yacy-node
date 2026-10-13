package vault

import (
	"errors"
	"fmt"
	"sync/atomic"
)

var (
	errProjectionFollowedTwice = errors.New("projection already follows a collection")
	errProjectionUnfollowed    = errors.New("projection follows no collection")
)

type Projection[R any] interface {
	projectionBucket
	recordAdded(tx *Txn, record R) error
	recordRemoved(tx *Txn, record R) error
	recordReplaced(tx *Txn, previous, current R) error
}

type projectionBucket interface {
	bucketName() Name
	follow() error
	staleIn(tx *Txn, versions *Collection[Name, string]) (bool, error)
	recordVersionIn(tx *Txn, versions *Collection[Name, string]) error
	clearFrom(tx *Txn, firstKey []byte, limit int) (nextKey []byte, err error)
}

type projectionRegistration struct {
	bucket   Name
	version  string
	followed atomic.Bool
}

func (r *projectionRegistration) bucketName() Name {
	return r.bucket
}

func (r *projectionRegistration) follow() error {
	if r.followed.Swap(true) {
		return fmt.Errorf("%w: %s", errProjectionFollowedTwice, r.bucket)
	}

	return nil
}

func (r *projectionRegistration) staleIn(
	tx *Txn,
	versions *Collection[Name, string],
) (bool, error) {
	storedVersion, found, err := versions.Get(tx, r.bucket)
	if err != nil {
		return false, err
	}

	return !found || storedVersion != r.version, nil
}

func (r *projectionRegistration) recordVersionIn(
	tx *Txn,
	versions *Collection[Name, string],
) error {
	_, err := versions.Put(tx, r.bucket, r.version)

	return err
}

func (r *projectionRegistration) refuseUnfollowed() error {
	if !r.followed.Load() {
		return fmt.Errorf("%w: %s", errProjectionUnfollowed, r.bucket)
	}

	return nil
}

type Projections[K, V any] interface {
	recordAdded(tx *Txn, key K, value V) error
	recordRemoved(tx *Txn, key K, value V) error
	recordReplaced(tx *Txn, key K, previous, current V) error
	buckets() []projectionBucket
	narrowedTo(buckets []projectionBucket) Projections[K, V]
}

func ProjectedAs[K, V, R any](
	formOf func(K, V) R,
	projections ...Projection[R],
) Projections[K, V] {
	return projectedForms[K, V, R]{formOf: formOf, projections: projections}
}

type projectedForms[K, V, R any] struct {
	formOf      func(K, V) R
	projections []Projection[R]
}

func (f projectedForms[K, V, R]) recordAdded(tx *Txn, key K, value V) error {
	record := f.formOf(key, value)
	for _, projection := range f.projections {
		if err := projection.recordAdded(tx, record); err != nil {
			return err
		}
	}

	return nil
}

func (f projectedForms[K, V, R]) recordRemoved(tx *Txn, key K, value V) error {
	record := f.formOf(key, value)
	for _, projection := range f.projections {
		if err := projection.recordRemoved(tx, record); err != nil {
			return err
		}
	}

	return nil
}

func (f projectedForms[K, V, R]) recordReplaced(tx *Txn, key K, previous, current V) error {
	previousRecord, currentRecord := f.formOf(key, previous), f.formOf(key, current)
	for _, projection := range f.projections {
		if err := projection.recordReplaced(tx, previousRecord, currentRecord); err != nil {
			return err
		}
	}

	return nil
}

func (f projectedForms[K, V, R]) buckets() []projectionBucket {
	buckets := make([]projectionBucket, 0, len(f.projections))
	for _, projection := range f.projections {
		buckets = append(buckets, projection)
	}

	return buckets
}

func (f projectedForms[K, V, R]) narrowedTo(buckets []projectionBucket) Projections[K, V] {
	narrowed := projectedForms[K, V, R]{formOf: f.formOf}
	for _, projection := range f.projections {
		for _, bucket := range buckets {
			if projectionBucket(projection) == bucket {
				narrowed.projections = append(narrowed.projections, projection)
			}
		}
	}

	return narrowed
}
