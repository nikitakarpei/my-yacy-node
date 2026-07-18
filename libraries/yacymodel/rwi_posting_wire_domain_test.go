package yacymodel

import (
	"testing"
	"time"
)

func mustParseDay(t *testing.T, day string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestRWIPostingWireFormDomainRoundTrip(t *testing.T) {
	original := RWIPosting{
		WordHash:               "ABCDEFGHIJKL",
		URLHash:                "MNOPQRSTUVWX",
		LastModified:           MicroDateFromTime(mustParseDay(t, "2026-07-18")),
		TitleWordCount:         3,
		TextWordCount:          120,
		PhraseCount:            8,
		DocType:                DocTypeImage,
		Language:               "en",
		LocalLinkCount:         2,
		ExternalLinkCount:      5,
		URLLength:              42,
		URLComponentCount:      4,
		Flags:                  AppearanceFlags{HasImage: true, AppearsInTitle: true},
		HitCount:               7,
		TextPosition:           258,
		PhraseRelativePosition: 3,
		PhrasePosition:         1,
	}

	wire := RWIPostingWireFormFromDomain(original)
	got, err := wire.Domain()
	if err != nil {
		t.Fatal(err)
	}
	if got != original {
		t.Fatalf("round trip = %+v, want %+v", got, original)
	}
}

func TestRWIPostingWireFormDomainFromSparseProperties(t *testing.T) {
	wire := RWIPostingWireForm{
		WordHash:   "ABCDEFGHIJKL",
		Properties: map[string]string{ColURLHash: "MNOPQRSTUVWX"},
	}

	got, err := wire.Domain()
	if err != nil {
		t.Fatal(err)
	}
	want := RWIPosting{WordHash: "ABCDEFGHIJKL", URLHash: "MNOPQRSTUVWX"}
	if got != want {
		t.Fatalf("Domain() = %+v, want %+v", got, want)
	}
}

func TestRWIPostingWireFormDomainRequiresURLHash(t *testing.T) {
	wire := RWIPostingWireForm{WordHash: "ABCDEFGHIJKL", Properties: map[string]string{}}
	if _, err := wire.Domain(); err == nil {
		t.Fatal("expected error for missing url hash")
	}
}

// TestStoredPostingPropertyFormDecodesToDomain locks in that a property-form
// line as it exists in already-stored data (real YaCy peers include the
// dead freshUntil/typeofword/worddistance/reserve columns) still parses and
// projects onto the domain type without error.
func TestStoredPostingPropertyFormDecodesToDomain(t *testing.T) {
	line := "ABCDEFGHIJKL{a=100,c=7,d=105,g=0,h=MNOPQRSTUVWX,i=0,k=0,l=en,m=42,n=4," +
		"o=1,p=8,r=3,s=100,t=258,u=3,w=120,x=2,y=5,z=AAAAAA}"

	wire, err := ParseRWIPosting(line)
	if err != nil {
		t.Fatal(err)
	}
	domain, err := wire.Domain()
	if err != nil {
		t.Fatal(err)
	}
	if domain.WordHash != "ABCDEFGHIJKL" || domain.URLHash != "MNOPQRSTUVWX" {
		t.Fatalf("Domain() hashes = %q/%q", domain.WordHash, domain.URLHash)
	}
	if domain.HitCount != 7 || domain.TextPosition != 258 {
		t.Fatalf("Domain() = %+v", domain)
	}

	// The dead columns present in the stored line are absent from the
	// re-encoded wire form; the property form itself (used by
	// stored_posting_form.go's binary encoder) is untouched by this
	// conversion and keeps decoding those columns directly from wire.
	if _, ok := wire.Properties[ColFreshUntil]; !ok {
		t.Fatal("wire form lost a stored column it did not itself touch")
	}
}
