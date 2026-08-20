package vault_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestARecordNoWriterOfThisVersionWroteRefusesTheRead(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		record []byte
	}{
		{name: "UnreadableMetadataLength", record: []byte{0xFF}},
		{name: "MetadataOfALaterWriter", record: []byte{0x01, 0x00, 'a'}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			engine := newDoubleEngine()

			v, err := vault.New(engine, nil)
			if err != nil {
				t.Fatalf("New: %v", err)
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
			engine.plant("words", stringKeyLayout.Key("a").Bytes(), testCase.record)

			if err := v.View(ctx, func(tx *vault.Txn) error {
				_, _, err := words.Get(tx, "a")

				return err
			}); err == nil {
				t.Fatal("Get of a record this writer cannot read succeeded, want error")
			}
		})
	}
}
