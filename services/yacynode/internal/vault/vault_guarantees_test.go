package vault_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var stringKeyLayout = vaultkey.Single(vaultkey.Text)

type stringKeyCodec struct{}

func (stringKeyCodec) Encode(key string) vaultkey.Key { return stringKeyLayout.Key(key) }

func (stringKeyCodec) Decode(key vaultkey.Key) (string, error) {
	decoded, err := stringKeyLayout.Parts(key)
	if err != nil {
		return "", fmt.Errorf("word key: %w", err)
	}

	return decoded, nil
}

type stringValueCodec struct{}

func (stringValueCodec) Encode(value string) ([]byte, error) { return []byte(value), nil }
func (stringValueCodec) Decode(raw []byte) (string, error)   { return string(raw), nil }

type failingEncodeCodec struct{}

func (failingEncodeCodec) Encode(
	string,
) ([]byte, error) {
	return nil, errors.New("encode boom")
}
func (failingEncodeCodec) Decode(raw []byte) (string, error) { return string(raw), nil }

type failingDecodeCodec struct{}

func (failingDecodeCodec) Encode(value string) ([]byte, error) { return []byte(value), nil }

func (failingDecodeCodec) Decode(
	[]byte,
) (string, error) {
	return "", errors.New("decode boom")
}

func openWords(t *testing.T) (*vault.Vault, *vault.Collection[string, string]) {
	t.Helper()

	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	words, err := vault.Register(v, vault.Name("words"), stringKeyCodec{}, stringValueCodec{})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	return v, words
}

func wrap(err error) error { return fmt.Errorf("vault op: %w", err) }

func TestPutThenGetTranslatesThroughCodec(t *testing.T) {
	ctx := context.Background()
	v, words := openWords(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		got, ok, err := words.Get(tx, "a")
		if err != nil {
			return wrap(err)
		}
		if !ok || got != "alpha" {
			t.Fatalf("Get(a) = %q, %v", got, ok)
		}

		length, err := words.Len(tx)
		if err != nil {
			return wrap(err)
		}
		if length != 1 {
			t.Fatalf("Len = %d, want 1", length)
		}

		return words.Scan(tx, vaultkey.EveryKey(), func(_ string, value string) (bool, error) {
			if value != "alpha" {
				t.Fatalf("Scan value = %q, want alpha", value)
			}

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func TestRejectsDuplicateRegistration(t *testing.T) {
	v, _ := openWords(t)

	if _, err := vault.Register(
		v,
		vault.Name("words"),
		stringKeyCodec{},
		stringValueCodec{},
	); err == nil {
		t.Fatal("duplicate Register succeeded, want error")
	}
}

func TestRegisterRejectsClosedVault(t *testing.T) {
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	if err := v.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := vault.Register(
		v,
		vault.Name("words"),
		stringKeyCodec{},
		stringValueCodec{},
	); err == nil {
		t.Fatal("Register on closed vault succeeded, want error")
	}
}

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

func TestEncodeErrorSurfaces(t *testing.T) {
	ctx := context.Background()
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	collection, err := vault.Register(
		v,
		vault.Name("words"),
		stringKeyCodec{},
		failingEncodeCodec{},
	)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return collection.Put(tx, "a", "alpha")
	}); err == nil {
		t.Fatal("Put with failing encode succeeded, want error")
	}
}

func TestDecodeErrorSurfaces(t *testing.T) {
	ctx := context.Background()
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	collection, err := vault.Register(
		v,
		vault.Name("words"),
		stringKeyCodec{},
		failingDecodeCodec{},
	)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return collection.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	getErr := v.View(ctx, func(tx *vault.Txn) error {
		_, _, err := collection.Get(tx, "a")
		if err != nil {
			return wrap(err)
		}

		return nil
	})
	if getErr == nil {
		t.Fatal("Get with failing decode succeeded, want error")
	}

	scanErr := v.View(ctx, func(tx *vault.Txn) error {
		return collection.Scan(
			tx,
			vaultkey.EveryKey(),
			func(string, string) (bool, error) { return true, nil },
		)
	})
	if scanErr == nil {
		t.Fatal("Scan with failing decode succeeded, want error")
	}
}

func TestAtCapacityIgnoresUnsetQuota(t *testing.T) {
	v, _ := openWords(t)

	atCapacity, err := v.AtCapacity(context.Background())
	if err != nil {
		t.Fatalf("AtCapacity: %v", err)
	}
	if atCapacity {
		t.Fatal("AtCapacity = true, want false without a quota")
	}
}

func TestClosedVaultRejectsOperations(t *testing.T) {
	ctx := context.Background()
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	if err := v.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := v.Update(ctx, func(*vault.Txn) error { return nil }); err == nil {
		t.Fatal("Update on closed vault succeeded, want error")
	}
	if err := v.View(ctx, func(*vault.Txn) error { return nil }); err == nil {
		t.Fatal("View on closed vault succeeded, want error")
	}
	if _, err := v.UsedBytes(ctx); err == nil {
		t.Fatal("UsedBytes on closed vault succeeded, want error")
	}
	if _, err := v.AtCapacity(ctx); err == nil {
		t.Fatal("AtCapacity on closed vault succeeded, want error")
	}
	if err := v.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
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
