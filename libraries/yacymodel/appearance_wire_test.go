package yacymodel

import "testing"

func TestAppearanceBitfieldRoundTrip(t *testing.T) {
	flags := Appearance{
		IndexOf:              true,
		HasVideo:             true,
		AppearsInTitle:       true,
		AppearsInIdentifier:  true,
		HasLocation:          false,
		HasImage:             false,
		HasAudio:             false,
		HasApp:               false,
		AppearsInDescription: false,
		AppearsInCreator:     false,
		AppearsInSubject:     false,
		Emphasized:           false,
	}

	got := AppearanceFromBitfield(flags.Bitfield())
	if got != flags {
		t.Fatalf("round trip = %+v, want %+v", got, flags)
	}
}

func TestAppearanceFromBitfieldIgnoresUnnamedBits(t *testing.T) {
	b := Bitfield{0, 0, 0, 0}
	b.setBit(1, true)
	b.setBit(30, true)

	got := AppearanceFromBitfield(b)
	if got != (Appearance{}) {
		t.Fatalf("AppearanceFromBitfield() = %+v, want zero value", got)
	}
}

func TestAppearanceBitfieldWidth(t *testing.T) {
	b := Appearance{}.Bitfield()
	if len(b) != appearanceFlagsByteWidth {
		t.Fatalf("Bitfield() length = %d, want %d", len(b), appearanceFlagsByteWidth)
	}
}
