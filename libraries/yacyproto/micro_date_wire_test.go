package yacyproto

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestMicroDateWireCodecRoundTrip(t *testing.T) {
	original := yacymodel.MicroDateFromTime(time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC))

	codec := microDateWireCodec{}
	if parsed := codec.decode(codec.encode(original)); parsed != original {
		t.Fatalf("round trip = %d, want %d", parsed, original)
	}
}

func TestMicroDateWireCodecDecodeWraps(t *testing.T) {
	if got := (microDateWireCodec{}).decode(microDateWireModulus); got != 0 {
		t.Fatalf("decode(%d) = %d, want 0", microDateWireModulus, got)
	}
}

func TestMicroDateWireCodecEncodeWrapsNegative(t *testing.T) {
	if got := (microDateWireCodec{}).encode(-1); got != microDateWireModulus-1 {
		t.Fatalf("encode(-1) = %d, want %d", got, microDateWireModulus-1)
	}
}
