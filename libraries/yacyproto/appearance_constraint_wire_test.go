package yacyproto

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestAppearanceConstraintWireCodecRoundTrip(t *testing.T) {
	want := yacymodel.Appearance{HasImage: true, AppearsInTitle: true}

	encoded := (appearanceConstraintWireCodec{}).encode(yacymodel.Some(want))
	got, err := (appearanceConstraintWireCodec{}).decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	appearance, ok := got.Get()
	if !ok || appearance != want {
		t.Fatalf("round trip = %+v, %v, want %+v", appearance, ok, want)
	}
}

// TestAppearanceConstraintWireCodecDecodesUnconstrained covers the three ways a
// peer says "no constraint". YaCy's own empty_constraint is an all-zero
// bitfield, which must not be read as "match nothing".
func TestAppearanceConstraintWireCodecDecodesUnconstrained(t *testing.T) {
	allSetBits := yacymodel.Encode([]byte{0xff, 0xff, 0xff, 0xff})
	for _, encoded := range []string{"", "AAAAAA", allSetBits} {
		got, err := (appearanceConstraintWireCodec{}).decode(encoded)
		if err != nil {
			t.Fatalf("decode(%q) error = %v", encoded, err)
		}
		if _, ok := got.Get(); ok {
			t.Errorf("decode(%q) is constrained, want unconstrained", encoded)
		}
	}
}

func TestAppearanceConstraintWireCodecEncodesNoneAsEmpty(t *testing.T) {
	got := (appearanceConstraintWireCodec{}).encode(yacymodel.None[yacymodel.Appearance]())
	if got != "" {
		t.Fatalf("encode(None) = %q, want empty", got)
	}
}

func TestAppearanceConstraintWireCodecRejectsBadEncoding(t *testing.T) {
	if _, err := (appearanceConstraintWireCodec{}).decode("AA=A"); err == nil {
		t.Fatal("expected error for malformed constraint")
	}
}
