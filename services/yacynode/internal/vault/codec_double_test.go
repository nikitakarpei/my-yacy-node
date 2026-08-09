package vault_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

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

	words, err := vault.RegisterCollection(
		v,
		vault.Name("words"),
		stringKeyCodec{},
		stringValueCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}

	return v, words
}

func wrap(err error) error { return fmt.Errorf("vault op: %w", err) }

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
