package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
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

func TestAfterCommitCallbackRunsOnceWhenTheWriteCommits(t *testing.T) {
	v, words := openWords(t)

	runs := 0
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		tx.RunAfterCommit(func() { runs++ })

		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if runs != 1 {
		t.Fatalf("runs = %d, want 1", runs)
	}
}

func TestAfterCommitCallbackStaysUnrunWhenTheWriteAborts(t *testing.T) {
	v, words := openWords(t)
	refused := errors.New("closure refused the write")

	runs := 0
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		tx.RunAfterCommit(func() { runs++ })
		if err := words.Put(tx, "a", "alpha"); err != nil {
			return wrap(err)
		}

		return refused
	}); !errors.Is(err, refused) {
		t.Fatalf("err = %v, want %v", err, refused)
	}

	if runs != 0 {
		t.Fatalf("runs = %d, want none: the write never committed", runs)
	}
}

func TestAfterCommitCallbackRunsOnceWhenTheEngineRepeatsTheClosure(t *testing.T) {
	v, err := vault.New(
		vaultenginetest.EngineRepeatingWrites(newDoubleEngine()),
		nil,
	)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	words, err := v.RegisterCollection(vault.Name("words"), stringKeyLayout, stringValueCodec{})
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}

	runs := 0
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		tx.RunAfterCommit(func() { runs++ })

		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if runs != 1 {
		t.Fatalf("runs = %d, want 1 even though the engine ran the closure twice", runs)
	}
}
