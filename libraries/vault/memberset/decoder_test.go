package memberset_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/memberset"
)

func TestADecoderReadsMembersOneAtATimeInStoredOrder(t *testing.T) {
	encoder := memberset.EncoderFrom(
		[]vault.StoredMember{{Raw: "zebra"}, {ID: 9}, {Raw: "yak"}, {ID: 2}},
	)
	decoder, _, err := memberset.DecoderFrom(encoder.AppendTo(nil))
	if err != nil {
		t.Fatalf("DecoderFrom: %v", err)
	}
	if amount := decoder.AmountOfMembers(); amount != 4 {
		t.Fatalf("amount of members = %d, want 4", amount)
	}

	var members []vault.StoredMember
	for member, found := decoder.Next(); found; member, found = decoder.Next() {
		members = append(members, member.Stored())
	}

	want := []vault.StoredMember{{ID: 2}, {ID: 9}, {Raw: "yak"}, {Raw: "zebra"}}
	if !slices.Equal(members, want) {
		t.Fatalf("members = %v, want %v", members, want)
	}
}

func TestDecodedMembersCompareInStoredOrder(t *testing.T) {
	for name, comparison := range map[string]struct {
		member, other memberset.DecodedMember
		want          int
	}{
		"lower id":      {memberset.DecodedMember{ID: 2}, memberset.DecodedMember{ID: 9}, -1},
		"same id":       {memberset.DecodedMember{ID: 9}, memberset.DecodedMember{ID: 9}, 0},
		"id before raw": {memberset.DecodedMember{ID: 9}, memberset.DecodedMember{Raw: []byte("a")}, -1},
		"raw after id":  {memberset.DecodedMember{Raw: []byte("a")}, memberset.DecodedMember{ID: 9}, 1},
		"higher raw":    {memberset.DecodedMember{Raw: []byte("zebra")}, memberset.DecodedMember{Raw: []byte("yak")}, 1},
	} {
		if got := comparison.member.CompareInStoredOrder(comparison.other); got != comparison.want {
			t.Errorf("%s: comparison = %d, want %d", name, got, comparison.want)
		}
	}
}
