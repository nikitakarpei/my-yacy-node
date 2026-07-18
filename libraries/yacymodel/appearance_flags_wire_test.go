package yacymodel

import "testing"

func TestAppearanceFlagsBitfieldRoundTrip(t *testing.T) {
	flags := AppearanceFlags{
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

	got := AppearanceFlagsFromBitfield(flags.Bitfield())
	if got != flags {
		t.Fatalf("round trip = %+v, want %+v", got, flags)
	}
}

func TestAppearanceFlagsFromBitfieldIgnoresUnnamedBits(t *testing.T) {
	b := Bitfield{0, 0, 0, 0}
	b.setBit(1, true)
	b.setBit(30, true)

	got := AppearanceFlagsFromBitfield(b)
	if got != (AppearanceFlags{}) {
		t.Fatalf("AppearanceFlagsFromBitfield() = %+v, want zero value", got)
	}
}

func TestAppearanceFlagsBitfieldWidth(t *testing.T) {
	b := AppearanceFlags{}.Bitfield()
	if len(b) != appearanceFlagsByteWidth {
		t.Fatalf("Bitfield() length = %d, want %d", len(b), appearanceFlagsByteWidth)
	}
}
