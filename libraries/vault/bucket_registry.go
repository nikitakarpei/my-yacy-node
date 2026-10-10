package vault

import (
	"context"
	"errors"
	"fmt"
)

var errDuplicateBucket = errors.New("bucket already registered")

type bucketKind int

const (
	recordsBucket bucketKind = iota
	amountsBucket
)

func (v *Vault) provisionRecordsBucket(bucket Name) error {
	if v == nil || v.engine == nil {
		return errVaultClosed
	}

	return v.provisionAs(bucket, recordsBucket, v.engine.ProvisionRecordsBucket)
}

func (v *Vault) provisionAs(bucket Name, kind bucketKind, provision func(Name) error) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, dup := v.registered[bucket]; dup {
		return fmt.Errorf("%w: %s", errDuplicateBucket, bucket)
	}

	if err := provision(bucket); err != nil {
		return fmt.Errorf("register bucket %s: %w", bucket, err)
	}

	v.registered[bucket] = kind

	return nil
}

func (v *Vault) provisionAmountsBucket(bucket Name) error {
	if v == nil || v.engine == nil {
		return errVaultClosed
	}

	return v.provisionAs(bucket, amountsBucket, v.engine.ProvisionAmountsBucket)
}

func (v *Vault) RecordCountsByBucket(ctx context.Context) (map[Name]int, error) {
	if v == nil || v.engine == nil {
		return nil, errVaultClosed
	}

	recordsBuckets := v.registeredRecordsBuckets()

	var recordCounts map[Name]int

	if err := v.view(ctx, func(tx *Txn) error {
		lengths, err := lengthsOf(tx, recordsBuckets)
		recordCounts = lengths

		return err
	}); err != nil {
		return nil, fmt.Errorf("read record counts: %w", err)
	}

	return recordCounts, nil
}

func lengthsOf(tx *Txn, recordsBuckets []Name) (map[Name]int, error) {
	lengths := make(map[Name]int, len(recordsBuckets))
	for _, bucket := range recordsBuckets {
		length, err := lengthOf(tx, bucket)
		if err != nil {
			return nil, err
		}
		lengths[bucket] = length
	}

	return lengths, nil
}

func (v *Vault) registeredRecordsBuckets() []Name {
	v.mu.Lock()
	defer v.mu.Unlock()

	recordsBuckets := make([]Name, 0, len(v.registered))
	for bucket, kind := range v.registered {
		if kind == recordsBucket {
			recordsBuckets = append(recordsBuckets, bucket)
		}
	}

	return recordsBuckets
}
