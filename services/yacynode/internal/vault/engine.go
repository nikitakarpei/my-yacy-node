package vault

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var ErrAtCapacity = errors.New("vault at capacity")

var (
	errVaultClosed            = errors.New("vault closed")
	errDuplicateBucket        = errors.New("bucket already registered")
	errReadOnly               = errors.New("write inside read-only transaction")
	errTransactionNeverOpened = errors.New("engine reported success without opening a transaction")
)

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
	Delete([]byte) error
	Scan(keys vaultkey.KeyRange, fn func(key, value []byte) (bool, error)) error
}
