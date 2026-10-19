package memberset_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/memberset"
)

func TestMembersReadBackAsAppendedWithTheBytesAfterThem(t *testing.T) {
	encoder := memberset.EncoderFrom([]vault.StoredMember{{ID: 9}, {Raw: "zebra"}, {ID: 2}})
	encoded := append(encoder.AppendTo([]byte("head")), "tail"...)

	members, rest, err := memberset.MembersFrom(encoded[len("head"):])
	if err != nil {
		t.Fatalf("MembersFrom: %v", err)
	}

	want := []vault.StoredMember{{ID: 2}, {ID: 9}, {Raw: "zebra"}}
	if !slices.Equal(members, want) || string(rest) != "tail" {
		t.Fatalf("members %v and rest %q, want %v and \"tail\"", members, rest, want)
	}
}

func TestMembersRefuseUnreadableEncodings(t *testing.T) {
	encoder := memberset.EncoderFrom([]vault.StoredMember{{ID: 3}, {Raw: "zebra"}})
	encoded := encoder.AppendTo(nil)

	for name, unreadable := range map[string][]byte{
		"no amount":       nil,
		"cut in ids":      encoded[:1],
		"cut in raws":     encoded[:len(encoded)-3],
		"raw of length 0": {0, 1, 0},
	} {
		if _, _, err := memberset.MembersFrom(unreadable); err == nil {
			t.Errorf("%s: MembersFrom succeeded, want refusal", name)
		}
	}
}
