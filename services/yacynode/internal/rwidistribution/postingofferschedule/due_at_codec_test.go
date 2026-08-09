package postingofferschedule

import (
	"bytes"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
)

func TestDueAtSurvivesTheValueRoundTrip(t *testing.T) {
	posting := postingidentity.IdentityOf(yacymodel.WordHash("w1"), urlHash("u1"))

	for name, dueAt := range map[string]time.Time{
		"zero time":      {},
		"recent instant": time.Unix(1700000000, 123456789).UTC(),
		"far future":     time.Date(2500, time.January, 1, 0, 0, 0, 0, time.UTC),
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := dueAtCodec{}.Encode(dueAt)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			decoded, err := dueAtCodec{}.Decode(raw)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			if !decoded.Equal(dueAt) {
				t.Fatalf("due at = %v, want %v", decoded, dueAt)
			}
			if !bytes.Equal(orderKeyFor(posting, decoded), orderKeyFor(posting, dueAt)) {
				t.Fatal(
					"the decoded due time addresses a different order row than the stored one",
				)
			}
		})
	}
}
