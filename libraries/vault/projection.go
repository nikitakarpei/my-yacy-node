package vault

type Projection[R any] interface {
	projectionBucket
	recordAdded(tx *Txn, added storedValue[R]) error
	recordRemoved(tx *Txn, removed storedValue[R]) error
	recordReplaced(tx *Txn, previous, current storedValue[R]) error
}

type projectionBucket interface {
	bucketName() Name
	attach() error
	registeredVersion() string
	clearFrom(tx *Txn, firstKey []byte, limit int) (nextKey []byte, err error)
}
