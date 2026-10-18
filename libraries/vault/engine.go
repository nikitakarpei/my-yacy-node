package vault

import (
	"context"
	"errors"
)

var ErrAtCapacity = errors.New("vault at capacity")

type Engine interface {
	Update(ctx context.Context, fn func(EngineTxn) error) error
	View(ctx context.Context, fn func(EngineTxn) error) error
	ProvisionRecordsBucket(Name) error
	ProvisionAmountsBucket(Name) error
	ProvisionMemberSetsBucket(Name) error
	UsedBytes(ctx context.Context) (int64, error)
	QuotaBytes() int64
	Close() error
}

type EngineTxn interface {
	Records(Name) EngineRecords
	Amounts(Name) EngineAmounts
	MemberSets(Name) EngineMemberSets
	Writable() bool
}

type EngineRecords interface {
	Get([]byte) ([]byte, error)
	Put(key, record []byte) (replacedRecord []byte, err error)
	Delete(key []byte) (deletedRecord []byte, err error)
	PutAbsent(key, record []byte) error
	DeleteHeld(key, heldRecord []byte) error
	Len() (int, error)
	Scan(keys KeyRange, fn func(key, value []byte) (bool, error)) error
}

type EngineAmounts interface {
	Get(key []byte) (int, error)
	Raise(key []byte, by uint) error
	Lower(key []byte, by uint) error
	Scan(keys KeyRange, fn func(key []byte, amount int) (bool, error)) error
}

type EngineMemberSets interface {
	Members(key []byte) ([]StoredMember, error)
	Add(key []byte, member StoredMember) error
	Remove(key []byte, member StoredMember) error
	Delete(key []byte) error
	Scan(keys KeyRange, fn func(key []byte, members []StoredMember) (bool, error)) error
}
