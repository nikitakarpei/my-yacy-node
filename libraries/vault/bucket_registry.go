package vault

import (
	"context"
	"errors"
	"fmt"
)

var errDuplicateBucket = errors.New("bucket already registered")

func (v *Vault) provision(bucket Name) error {
	if v == nil || v.engine == nil {
		return errVaultClosed
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	if _, dup := v.registered[bucket]; dup {
		return fmt.Errorf("%w: %s", errDuplicateBucket, bucket)
	}

	if err := v.engine.Provision(bucket); err != nil {
		return fmt.Errorf("register bucket %s: %w", bucket, err)
	}

	v.registered[bucket] = struct{}{}

	return nil
}

func (v *Vault) EntriesByCollection(ctx context.Context) (map[Name]int, error) {
	if v == nil || v.engine == nil {
		return nil, errVaultClosed
	}

	collections := v.registeredCollections()

	var entries map[Name]int

	if err := v.view(ctx, func(tx *Txn) error {
		lengths, err := lengthsOf(tx, collections)
		entries = lengths

		return err
	}); err != nil {
		return nil, fmt.Errorf("read collection entries: %w", err)
	}

	return entries, nil
}

func (v *Vault) registeredCollections() []Name {
	v.mu.Lock()
	defer v.mu.Unlock()

	collections := make([]Name, 0, len(v.registered))
	for bucket := range v.registered {
		collections = append(collections, bucket)
	}

	return collections
}
