package vault_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestAMemberSetChangeReadsBackAsEncoded(t *testing.T) {
	change := vault.MemberSetChange{
		Added:   []vault.StoredMember{{ID: 9}, {Raw: "zebra"}, {ID: 2}},
		Removed: []vault.StoredMember{{Raw: "yak"}, {ID: 0}},
	}

	decoded, err := vault.MemberSetChangeFrom(change.Encoded())
	if err != nil {
		t.Fatalf("MemberSetChangeFrom: %v", err)
	}

	wantAdded := []vault.StoredMember{{ID: 2}, {ID: 9}, {Raw: "zebra"}}
	wantRemoved := []vault.StoredMember{{ID: 0}, {Raw: "yak"}}
	if !slices.Equal(decoded.Added, wantAdded) || !slices.Equal(decoded.Removed, wantRemoved) {
		t.Fatalf("decoded = %+v, want added %v and removed %v", decoded, wantAdded, wantRemoved)
	}
}

func TestAMemberSetChangeRefusesUnreadableEncodings(t *testing.T) {
	encoded := vault.MemberSetChange{
		Added: []vault.StoredMember{{ID: 3}, {Raw: "zebra"}},
	}.Encoded()

	for name, unreadable := range map[string][]byte{
		"empty":          nil,
		"unknown format": append([]byte{0x7f}, encoded[1:]...),
		"cut short":      encoded[:len(encoded)-3],
		"trailing bytes": append(slices.Clone(encoded), 0x00),
		"missing count":  encoded[:1],
		"cut id":         {encoded[0], 0x01, 0x80},
		"empty raw":      {encoded[0], 0x00, 0x01, 0x00, 0x00, 0x00},
	} {
		if _, err := vault.MemberSetChangeFrom(unreadable); err == nil {
			t.Errorf("%s: MemberSetChangeFrom succeeded, want refusal", name)
		}
	}
}
