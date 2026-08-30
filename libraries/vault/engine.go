package vault

import (
	"context"
	"errors"
)

var ErrAtCapacity = errors.New("vault at capacity")

type Engine interface {
	Update(ctx context.Context, fn func(EngineTxn) error) error
	View(ctx context.Context, fn func(EngineTxn) error) error
	Provision(Name) error
	UsedBytes(ctx context.Context) (int64, error)
	QuotaBytes() int64
	Close() error
}

type EngineTxn interface {
	Bucket(Name) EngineBucket
	Writable() bool
}

type EngineBucket interface {
	Get([]byte) []byte
	Put([]byte, []byte) error
	Delete([]byte) (bool, error)
	Len() (int, error)
	Scan(keys KeyRange, fn func(key, value []byte) (bool, error)) error
}
