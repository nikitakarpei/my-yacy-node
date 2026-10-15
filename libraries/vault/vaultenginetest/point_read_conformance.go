package vaultenginetest

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func runPointReadConformance(t *testing.T, open func(int64) (vault.Engine, error)) {
	t.Helper()

	t.Run(
		"ReadsInOneViewFindEachBucketsRecordsInAnyOrder",
		func(t *testing.T) { readsInOneViewFindEachBucketsRecordsInAnyOrder(t, open) },
	)
}

type pointRead struct {
	collection *vault.Collection[string, string]
	key        string
	value      string
	held       bool
}

func readsInOneViewFindEachBucketsRecordsInAnyOrder(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	ctx := context.Background()
	v := openVault(t, open, 0)
	words, pages := register(t, v, "words"), register(t, v, "pages")
	if err := v.Update(ctx, func(tx *vault.Txn) error {
		for _, stored := range []pointRead{
			{collection: words, key: "a", value: "alpha"},
			{collection: words, key: "ab", value: "alphabet"},
			{collection: words, key: "c", value: "gamma"},
			{collection: words, key: "d", value: "delta"},
			{collection: pages, key: "b", value: "page b"},
			{collection: pages, key: "c", value: "page c"},
		} {
			if _, err := stored.collection.Put(tx, stored.key, stored.value); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	reads := []pointRead{
		{collection: words, key: "d", value: "delta", held: true},
		{collection: pages, key: "b", value: "page b", held: true},
		{collection: words, key: "a", value: "alpha", held: true},
		{collection: pages, key: "a"},
		{collection: words, key: "b"},
		{collection: words, key: "ab", value: "alphabet", held: true},
		{collection: pages, key: "c", value: "page c", held: true},
		{collection: words, key: "z"},
		{collection: words, key: "c", value: "gamma", held: true},
		{collection: words, key: "a", value: "alpha", held: true},
	}
	if err := v.View(ctx, func(tx *vault.Txn) error {
		for _, read := range reads {
			got, found, err := read.collection.Get(tx, read.key)
			if err != nil {
				return wrapTest(err)
			}
			if got != read.value || found != read.held {
				t.Errorf(
					"Get(%q) = %q, %v, want %q, %v",
					read.key,
					got,
					found,
					read.value,
					read.held,
				)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}
