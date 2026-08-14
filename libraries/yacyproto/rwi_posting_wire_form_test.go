package yacyproto_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func postingURLHash(t *testing.T) yacymodel.URLHash {
	t.Helper()
	hash, err := yacymodel.ParseURLHash("MNOPQRSTUVWX")
	if err != nil {
		t.Fatal(err)
	}

	return hash
}

func mustParseDay(t *testing.T, day string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func postingFromLine(t *testing.T, line string) yacymodel.RWIPosting {
	t.Helper()

	form := url.Values{yacyproto.FieldIndexes: {line}}
	req, err := yacyproto.ParseTransferRWIRequest(t.Context(), form)
	if err != nil {
		t.Fatalf("ParseTransferRWIRequest: %v", err)
	}
	if len(req.Indexes) != 1 {
		t.Fatalf("Indexes = %d, want 1 for line %q", len(req.Indexes), line)
	}

	return req.Indexes[0]
}

func postingRoundTrip(t *testing.T, posting yacymodel.RWIPosting) yacymodel.RWIPosting {
	t.Helper()

	line := yacyproto.TransferRWIRequest{
		Indexes: []yacymodel.RWIPosting{posting},
	}.Form().Get(yacyproto.FieldIndexes)

	return postingFromLine(t, line)
}

func TestTransferRWIRequestCarriesEveryPostingColumn(t *testing.T) {
	t.Parallel()

	want := yacymodel.RWIPosting{
		WordHash:               mustHash(t, "ABCDEFGHIJKL"),
		URLHash:                postingURLHash(t),
		LastModified:           yacymodel.MicroDateFromTime(mustParseDay(t, "2026-07-18")),
		TitleWords:             3,
		TextWords:              120,
		Phrases:                8,
		DocumentType:           yacymodel.DocumentTypeImage,
		Language:               mustLanguage(t, "en"),
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

	if got := postingRoundTrip(t, want); got != want {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}

func TestTransferRWIRequestReadsALineAsARealPeerSendsIt(t *testing.T) {
	t.Parallel()

	line := "ABCDEFGHIJKL{a=100,c=7,d=105,g=0,h=MNOPQRSTUVWX,i=0,k=0,l=en,m=42,n=4," +
		"o=1,p=8,r=3,s=100,t=258,u=3,w=120,x=2,y=5,z=AAAAAA}"

	got := postingFromLine(t, line)
	if got.WordHash.String() != "ABCDEFGHIJKL" || got.URLHash.String() != "MNOPQRSTUVWX" {
		t.Fatalf("hashes = %q/%q", got.WordHash, got.URLHash)
	}
	if got.Hits != 7 || got.TextPosition != 258 || got.DocumentType != yacymodel.DocumentTypeImage {
		t.Fatalf("posting = %+v", got)
	}
}

func TestTransferRWIRequestNormalizesYaCyPropertyForm(t *testing.T) {
	t.Parallel()

	got := postingFromLine(t, "ABCDEFGHIJKL{c=1,d=104,h=MNOPQRSTUVWX,l=eng,t=258x,x=2,z=AAAAAAA}")
	if got.Hits != 1 || got.TextPosition != 258 || got.LocalLinks != 2 {
		t.Fatalf("cardinals = %+v", got)
	}
	if language, ok := got.Language.Get(); !ok || language.String() != "en" {
		t.Fatalf("language = %v, want %q", got.Language, "en")
	}
	if got.DocumentType != yacymodel.DocumentTypeHTML {
		t.Fatalf("document type = %v", got.DocumentType)
	}
}

func TestTransferRWIRequestKeepsALastModifiedDateWiderThanTwoBytes(t *testing.T) {
	t.Parallel()

	got := postingFromLine(t, "ABCDEFGHIJKL{a=200000,h=MNOPQRSTUVWX}")
	if got.LastModified != yacymodel.MicroDate(200000) {
		t.Fatalf("last modified = %d, want 200000", got.LastModified)
	}
}

func TestTransferRWIRequestWrapsTheLastModifiedDateAtTheYaCyModulus(t *testing.T) {
	t.Parallel()

	const modulus = 262144
	cases := map[yacymodel.MicroDate]yacymodel.MicroDate{
		modulus:     0,
		-1:          modulus - 1,
		modulus + 5: 5,
	}
	for written, want := range cases {
		posting := yacymodel.RWIPosting{
			WordHash:     mustHash(t, "ABCDEFGHIJKL"),
			URLHash:      postingURLHash(t),
			LastModified: written,
		}
		if got := postingRoundTrip(t, posting).LastModified; got != want {
			t.Errorf("last modified %d round trips to %d, want %d", written, got, want)
		}
	}
}

func TestTransferRWIRequestReadsASparseLine(t *testing.T) {
	t.Parallel()

	want := yacymodel.RWIPosting{
		WordHash: mustHash(t, "ABCDEFGHIJKL"),
		URLHash:  postingURLHash(t),
	}
	if got := postingFromLine(t, "ABCDEFGHIJKL{h=MNOPQRSTUVWX}"); got != want {
		t.Fatalf("posting = %+v, want %+v", got, want)
	}
}
