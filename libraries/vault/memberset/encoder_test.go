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

func TestAnEncoderForDecodersCountsTheBytesItAppends(t *testing.T) {
	stored := memberset.EncoderFrom([]vault.StoredMember{{ID: 300}, {ID: 2}, {Raw: "zebra"}})
	decoder, _, err := memberset.DecoderFrom(stored.AppendTo(nil))
	if err != nil {
		t.Fatalf("DecoderFrom: %v", err)
	}

	encoder := memberset.EncoderFor([]memberset.Decoder{decoder})
	for member, found := decoder.Next(); found; member, found = decoder.Next() {
		encoder.Add(member)
	}

	encoded := encoder.AppendTo(nil)
	if amount := encoder.AmountOfBytes(); amount != len(encoded) {
		t.Fatalf("amount of bytes = %d, want %d", amount, len(encoded))
	}
	members, _, err := memberset.MembersFrom(encoded)
	if err != nil {
		t.Fatalf("MembersFrom: %v", err)
	}
	want := []vault.StoredMember{{ID: 2}, {ID: 300}, {Raw: "zebra"}}
	if !slices.Equal(members, want) {
		t.Fatalf("members = %v, want %v", members, want)
	}
}
