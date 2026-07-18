package yacyproto

import (
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func mustParseDay(t *testing.T, day string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestRWIPostingWireCodecRoundTrip(t *testing.T) {
	original := yacymodel.RWIPosting{
		WordHash:               "ABCDEFGHIJKL",
		URLHash:                "MNOPQRSTUVWX",
		LastModified:           yacymodel.MicroDateFromTime(mustParseDay(t, "2026-07-18")),
		TitleWords:             3,
		TextWords:              120,
		Phrases:                8,
		DocumentType:           yacymodel.DocumentTypeImage,
		Language:               "en",
		LocalLinks:             2,
		ExternalLinks:          5,
		URLLength:              42,
		URLComponents:          4,
		Appearance:             yacymodel.Appearance{HasImage: true, AppearsInTitle: true},
		Hits:                   7,
		TextPosition:           258,
		PhraseRelativePosition: 3,
		PhrasePosition:         1,
	}

	got, err := rwiPostingWireCodec{}.decode(rwiPostingWireCodec{}.encode(original))
	if err != nil {
		t.Fatal(err)
	}
	if got != original {
		t.Fatalf("round trip = %+v, want %+v", got, original)
	}
}

// TestRWIPostingWireCodecDecodesStoredLine locks in that a property-form line
// as a real YaCy peer sends it -- including the freshUntil, typeofword,
// worddistance and reserve columns this node has no use for -- still decodes.
func TestRWIPostingWireCodecDecodesStoredLine(t *testing.T) {
	line := "ABCDEFGHIJKL{a=100,c=7,d=105,g=0,h=MNOPQRSTUVWX,i=0,k=0,l=en,m=42,n=4," +
		"o=1,p=8,r=3,s=100,t=258,u=3,w=120,x=2,y=5,z=AAAAAA}"

	got, err := rwiPostingWireCodec{}.decode(line)
	if err != nil {
		t.Fatal(err)
	}
	if got.WordHash != "ABCDEFGHIJKL" || got.URLHash != "MNOPQRSTUVWX" {
		t.Fatalf("decode() hashes = %q/%q", got.WordHash, got.URLHash)
	}
	if got.Hits != 7 || got.TextPosition != 258 || got.DocumentType != yacymodel.DocumentTypeImage {
		t.Fatalf("decode() = %+v", got)
	}
}

func TestRWIPostingWireCodecDecodeNormalizesYaCyPropertyForm(t *testing.T) {
	got, err := rwiPostingWireCodec{}.decode(
		"ABCDEFGHIJKL{c=1,d=104,h=MNOPQRSTUVWX,l=eng,t=258x,x=2,z=AAAAAAA}",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hits != 1 || got.TextPosition != 258 || got.LocalLinks != 2 {
		t.Fatalf("decode() cardinals = %+v", got)
	}
	if got.Language != "en" {
		t.Fatalf("decode() language = %q, want %q", got.Language, "en")
	}
	if got.DocumentType != yacymodel.DocumentTypeHTML {
		t.Fatalf("decode() document type = %v", got.DocumentType)
	}
}

// TestRWIPostingWireCodecDecodeKeepsWideDate covers a day count above 65535.
// The last modified column wraps at YaCy's 64**3 day modulus, not at the two
// bytes the other wide columns use.
func TestRWIPostingWireCodecDecodeKeepsWideDate(t *testing.T) {
	got, err := rwiPostingWireCodec{}.decode("ABCDEFGHIJKL{a=200000,h=MNOPQRSTUVWX}")
	if err != nil {
		t.Fatal(err)
	}
	if got.LastModified != yacymodel.MicroDate(200000) {
		t.Fatalf("decode() last modified = %d, want 200000", got.LastModified)
	}
}

func TestRWIPostingWireCodecDecodeSparseLine(t *testing.T) {
	got, err := rwiPostingWireCodec{}.decode("ABCDEFGHIJKL{h=MNOPQRSTUVWX}")
	if err != nil {
		t.Fatal(err)
	}
	want := yacymodel.RWIPosting{WordHash: "ABCDEFGHIJKL", URLHash: "MNOPQRSTUVWX"}
	if got != want {
		t.Fatalf("decode() = %+v, want %+v", got, want)
	}
}

func TestRWIPostingWireCodecDecodeErrors(t *testing.T) {
	cases := []string{
		"ABCDEFGHIJKLnobraces",
		"short{h=MNOPQRSTUVWX}",
		"ABCDEFGHIJKL{h=MNOPQRSTUVWX,badtoken}",
		"ABCDEFGHIJKL{h=notahash}",
		"ABCDEFGHIJKL{}",
	}
	for _, c := range cases {
		_, err := (rwiPostingWireCodec{}).decode(c)
		if !errors.Is(err, yacymodel.ErrBadRWIPosting) {
			t.Errorf("decode(%q) = %v, want ErrBadRWIPosting", c, err)
		}
	}
}
