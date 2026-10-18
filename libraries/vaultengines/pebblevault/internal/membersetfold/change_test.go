package membersetfold_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault/internal/membersetfold"
)

func TestAChangeReadsBackAsEncoded(t *testing.T) {
	change := membersetfold.Change{
		Added:   []vault.StoredMember{{ID: 9}, {Raw: "zebra"}, {ID: 2}},
		Removed: []vault.StoredMember{{Raw: "yak"}, {ID: 0}},
	}

	decoded, err := membersetfold.ChangeFrom(change.Encoded())
	if err != nil {
		t.Fatalf("ChangeFrom: %v", err)
	}

	wantAdded := []vault.StoredMember{{ID: 2}, {ID: 9}, {Raw: "zebra"}}
	wantRemoved := []vault.StoredMember{{ID: 0}, {Raw: "yak"}}
	if !slices.Equal(decoded.Added, wantAdded) || !slices.Equal(decoded.Removed, wantRemoved) {
		t.Fatalf("decoded = %+v, want added %v and removed %v", decoded, wantAdded, wantRemoved)
	}
}

func TestAChangeRefusesUnreadableEncodings(t *testing.T) {
	encoded := membersetfold.Change{
		Added:   []vault.StoredMember{{ID: 3}, {Raw: "zebra"}},
		Removed: []vault.StoredMember{{ID: 4}},
	}.Encoded()

	for name, unreadable := range map[string][]byte{
		"unknown format": append([]byte{0x7f}, encoded[1:]...),
		"cut in added":   encoded[:5],
		"cut in removed": encoded[:len(encoded)-2],
		"trailing bytes": append(slices.Clone(encoded), 0),
	} {
		if _, err := membersetfold.ChangeFrom(unreadable); err == nil {
			t.Errorf("%s: ChangeFrom succeeded, want refusal", name)
		}
	}
}
