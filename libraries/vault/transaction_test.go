package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestWriteInsideViewReturnsError(t *testing.T) {
	ctx := context.Background()
	v, words := openWords(t)

	putErr := v.View(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	})
	if putErr == nil {
		t.Fatal("Put inside View succeeded, want error")
	}

	deleteErr := v.View(ctx, func(tx *vault.Txn) error {
		_, delErr := words.Delete(tx, "a")
		if delErr != nil {
			return wrap(delErr)
		}

		return nil
	})
	if deleteErr == nil {
		t.Fatal("Delete inside View succeeded, want error")
	}
}

func TestCancelledContextStopsTransactions(t *testing.T) {
	v, _ := openWords(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := v.Update(
		ctx,
		func(*vault.Txn) error { return nil },
	); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("Update err = %v, want context.Canceled", err)
	}
	if err := v.View(
		ctx,
		func(*vault.Txn) error { return nil },
	); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("View err = %v, want context.Canceled", err)
	}
}
