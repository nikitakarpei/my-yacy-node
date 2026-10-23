package vault

type Projections[K, V any] interface {
	recordAdded(tx *Txn, key K, added storedValue[V]) error
	recordRemoved(tx *Txn, key K, removed storedValue[V]) error
	recordReplaced(tx *Txn, key K, previous, current storedValue[V]) error
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

func (f projectedForms[K, V, R]) recordAdded(tx *Txn, key K, added storedValue[V]) error {
	projected := f.projectedFrom(key, added)
	for _, projection := range f.projections {
		if err := projection.recordAdded(tx, projected); err != nil {
			return err
		}
	}

	return nil
}

func (f projectedForms[K, V, R]) projectedFrom(key K, stored storedValue[V]) storedValue[R] {
	return storedValue[R]{value: f.formOf(key, stored.value), encoded: stored.encoded}
}

func (f projectedForms[K, V, R]) recordRemoved(tx *Txn, key K, removed storedValue[V]) error {
	projected := f.projectedFrom(key, removed)
	for _, projection := range f.projections {
		if err := projection.recordRemoved(tx, projected); err != nil {
			return err
		}
	}

	return nil
}

func (f projectedForms[K, V, R]) recordReplaced(
	tx *Txn,
	key K,
	previous, current storedValue[V],
) error {
	projectedPrevious, projectedCurrent := f.projectedFrom(
		key,
		previous,
	), f.projectedFrom(
		key,
		current,
	)
	for _, projection := range f.projections {
		if err := projection.recordReplaced(tx, projectedPrevious, projectedCurrent); err != nil {
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
