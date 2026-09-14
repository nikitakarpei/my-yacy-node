package yacyproto_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	postingWordHash = "ABCDEFGHIJKL"
	postingURLHash  = "MNOPQRSTUVWX"
)

func mustPostingURLHash(t *testing.T) yacymodel.URLHash {
	t.Helper()
	hash, err := yacymodel.ParseURLHash(postingURLHash)
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
		WordHash:               mustHash(t, postingWordHash),
		URLHash:                mustPostingURLHash(t),
		LastModified:           yacymodel.MicroDateFromTime(mustParseDay(t, "2026-07-18")),
		TitleWords:             3,
		TextWords:              120,
		Phrases:                8,
		DocumentType:           yacymodel.DocumentTypeImage,
		Language:               englishLanguage(t),
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

	line := postingWordHash + "{a=100,c=7,d=105,g=0,h=" + postingURLHash + ",i=0,k=0,l=en,m=42," +
		"n=4,o=1,p=8,r=3,s=100,t=258,u=3,w=120,x=2,y=5,z=AAAAAA}"

	got := postingFromLine(t, line)
	if got.WordHash.String() != postingWordHash || got.URLHash.String() != postingURLHash {
		t.Fatalf("hashes = %q/%q", got.WordHash, got.URLHash)
	}
	if got.Hits != 7 || got.TextPosition != 258 || got.DocumentType != yacymodel.DocumentTypeImage {
		t.Fatalf("posting = %+v", got)
	}
}

func TestTransferRWIRequestNormalizesYaCyPropertyForm(t *testing.T) {
	t.Parallel()

	line := postingWordHash + "{c=1,d=104,h=" + postingURLHash + ",l=eng,t=258x,x=2,z=AAAAAAA}"
	got := postingFromLine(t, line)
	if got.Hits != 1 || got.TextPosition != 258 || got.LocalLinks != 2 {
		t.Fatalf("cardinals = %+v", got)
	}
	if got.Language.String() != "en" {
		t.Fatalf("language = %v, want %q", got.Language, "en")
	}
	if got.DocumentType != yacymodel.DocumentTypeHTML {
		t.Fatalf("document type = %v", got.DocumentType)
	}
}

func TestTransferRWIRequestKeepsALastModifiedDateWiderThanTwoBytes(t *testing.T) {
	t.Parallel()

	line := postingWordHash + "{a=200000,h=" + postingURLHash + ",l=en}"
	got := postingFromLine(t, line)
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
			WordHash:     mustHash(t, postingWordHash),
			URLHash:      mustPostingURLHash(t),
			Language:     englishLanguage(t),
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
		WordHash: mustHash(t, postingWordHash),
		URLHash:  mustPostingURLHash(t),
		Language: englishLanguage(t),
	}
	if got := postingFromLine(t, postingWordHash+"{h="+postingURLHash+",l=en}"); got != want {
		t.Fatalf("posting = %+v, want %+v", got, want)
	}
}

func TestTransferRWIRequestRejectsALineWithoutALanguage(t *testing.T) {
	t.Parallel()

	rejectedLines := map[string]string{
		"absent":      postingWordHash + "{h=" + postingURLHash + "}",
		"empty":       postingWordHash + "{h=" + postingURLHash + ",l=}",
		"zero bytes":  postingWordHash + "{h=" + postingURLHash + ",l=\x00\x00}",
		"not letters": postingWordHash + "{h=" + postingURLHash + ",l=12}",
	}
	for name, line := range rejectedLines {
		form := url.Values{yacyproto.FieldIndexes: {line}}
		req, err := yacyproto.ParseTransferRWIRequest(t.Context(), form)
		if err != nil {
			t.Fatalf("ParseTransferRWIRequest: %v", err)
		}
		if len(req.Indexes) != 0 {
			t.Errorf("%s language kept %+v, want a rejected posting", name, req.Indexes)
		}
	}
}

func postingFromCardinals(t *testing.T, byteCardinal, uint16Cardinal int) yacymodel.RWIPosting {
	t.Helper()

	return yacymodel.RWIPosting{
		WordHash:               mustHash(t, postingWordHash),
		URLHash:                mustPostingURLHash(t),
		Language:               englishLanguage(t),
		TitleWords:             byteCardinal,
		LocalLinks:             byteCardinal,
		ExternalLinks:          byteCardinal,
		URLLength:              byteCardinal,
		URLComponents:          byteCardinal,
		Hits:                   byteCardinal,
		PhraseRelativePosition: byteCardinal,
		PhrasePosition:         byteCardinal,
		TextWords:              uint16Cardinal,
		Phrases:                uint16Cardinal,
		TextPosition:           uint16Cardinal,
	}
}

func TestTransferRWIRequestSaturatesCardinalsWiderThanTheirColumn(t *testing.T) {
	t.Parallel()

	const (
		byteCeiling   = 255
		uint16Ceiling = 65535
	)
	saturated := postingFromCardinals(t, byteCeiling, uint16Ceiling)
	cases := map[string]struct {
		written, want yacymodel.RWIPosting
	}{
		"at the ceiling": {saturated, saturated},
		"one above the ceiling": {
			postingFromCardinals(t, byteCeiling+1, uint16Ceiling+1),
			saturated,
		},
		"far above the ceiling": {postingFromCardinals(t, 4096, 1<<20), saturated},
		"below zero": {
			postingFromCardinals(t, -1, -300),
			postingFromCardinals(t, 0, 0),
		},
	}
	for name, testCase := range cases {
		if got := postingRoundTrip(t, testCase.written); got != testCase.want {
			t.Errorf("%s round trips to %+v, want %+v", name, got, testCase.want)
		}
	}
}
