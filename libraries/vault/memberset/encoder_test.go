package memberset_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/memberset"
)

func TestMembersAddedInStoredOrderReadBackAsAdded(t *testing.T) {
	var encoder memberset.Encoder
	for _, member := range []memberset.DecodedMember{{ID: 2}, {ID: 9}, {Raw: []byte("yak")}} {
		encoder.Add(member)
	}

	members, rest, err := memberset.MembersFrom(encoder.AppendTo(nil))
	if err != nil {
		t.Fatalf("MembersFrom: %v", err)
	}

	want := []vault.StoredMember{{ID: 2}, {ID: 9}, {Raw: "yak"}}
	if !slices.Equal(members, want) || len(rest) != 0 {
		t.Fatalf("members %v and rest %q, want %v and nothing", members, rest, want)
	}
}

func TestAnEncoderCountsTheBytesItAppends(t *testing.T) {
	encoder := memberset.EncoderFrom([]vault.StoredMember{{ID: 300}, {ID: 2}, {Raw: "zebra"}})

	if amount, encoded := encoder.AmountOfBytes(), encoder.AppendTo(nil); amount != len(encoded) {
		t.Fatalf("amount of bytes = %d, want %d", amount, len(encoded))
	}
}
