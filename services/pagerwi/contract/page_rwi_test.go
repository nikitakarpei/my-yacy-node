package pagerwicontract_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
)

const pageAddress = "https://example.org/page"

func TestPageRWIRoundTripsEveryFact(t *testing.T) {
	built := pageRWIWithEveryFact(t)

	data, err := pagerwicontract.MarshalPageRWI(built)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	read, err := pagerwicontract.UnmarshalPageRWI(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(read, built) {
		t.Errorf("read rwi = %+v, want %+v", read, built)
	}
}

func TestPageRWIWithoutPostingsRoundTripsWithoutPostings(t *testing.T) {
	built := pagerwicontract.PageRWI{
		PageURL: canonicalurltest.CanonicalURLOf(t, pageAddress),
		Metadata: yacymodel.URLMetadata{
			Hash:    urlhashtest.URLHashOf(t, pageAddress),
			Address: pageAddress,
		},
	}

	data, err := pagerwicontract.MarshalPageRWI(built)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	read, err := pagerwicontract.UnmarshalPageRWI(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(read.Postings) != 0 || read.Metadata.PostingsRoot.Present() {
		t.Errorf("read rwi = %+v, want no postings and no postings root", read)
	}
}

func TestUnmarshalPageRWIRejectsAnUndecodablePayload(t *testing.T) {
	for name, data := range map[string]string{
		"no json":       "not an rwi",
		"bad url hash":  `{"Metadata":{"Hash":"short"}}`,
		"bad word hash": `{"Postings":[{"WordHash":"!"}]}`,
		"bad root":      `{"Metadata":{"PostingsRoot":"AAAA"}}`,
		"bad language":  `{"Postings":[{"Language":"english"}]}`,
		"bad page url":  `{"PageURL":"not a url"}`,
	} {
		if _, err := pagerwicontract.UnmarshalPageRWI([]byte(data)); err == nil {
			t.Errorf("%s: unmarshal accepted %s", name, data)
		}
	}
}

func pageRWIWithEveryFact(t *testing.T) pagerwicontract.PageRWI {
	t.Helper()

	urlHash := urlhashtest.URLHashOf(t, pageAddress)
	postings := []yacymodel.RWIPosting{
		postingOf(t, "alpha", urlHash),
		postingOf(t, "beta", urlHash),
	}
	pagePostings, err := yacymodel.ProvenPagePostingsFrom(
		postings, yacymodel.EmptyPostingsProof(partitionsOf(t, 4)),
	)
	if err != nil {
		t.Fatalf("page postings: %v", err)
	}
	location, err := yacymodel.NewCoordinates(52.5, 13.4)
	if err != nil {
		t.Fatalf("coordinates: %v", err)
	}

	return pagerwicontract.PageRWI{
		PageURL: canonicalurltest.CanonicalURLOf(t, pageAddress),
		Metadata: yacymodel.URLMetadata{
			Hash:          urlHash,
			Referrer:      yacymodel.Some(urlhashtest.URLHashOf(t, "https://example.org/")),
			DocumentType:  yacymodel.DocumentTypeText,
			Language:      yacymodel.Some(languageOf(t, "de")),
			PostingsRoot:  pagePostings.Root(),
			ByteSize:      yacymodel.Some(uint32(2048)),
			WordCount:     2,
			LocalLinks:    3,
			ExternalLinks: 4,
			Address:       pageAddress,
			Title:         "Alpha and beta",
			Tags:          []string{"one", "two"},
			Location:      yacymodel.Some(location),
			Modified:      yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, 30)),
			Loaded:        yacymodel.Some(yacymodel.NewCalendarDay(2026, time.November, 6)),
			MediaType:     "text/html",
		},
		Postings: pagePostings.Postings(),
	}
}

func postingOf(t *testing.T, word string, urlHash yacymodel.URLHash) yacymodel.RWIPosting {
	t.Helper()

	return yacymodel.RWIPosting{
		WordHash:     yacymodel.WordHash(word),
		URLHash:      urlHash,
		LastModified: 1234,
		Language:     languageOf(t, "de"),
		DocumentType: yacymodel.DocumentTypeText,
		TitleWords:   3,
		TextWords:    2,
		Phrases:      1,
		Appearance:   yacymodel.Appearance{AppearsInTitle: true},
		TextPosition: 1,
		Hits:         1,
	}
}

func languageOf(t *testing.T, code string) yacymodel.Language {
	t.Helper()

	language, err := yacymodel.ParseLanguage(code)
	if err != nil {
		t.Fatalf("language %q: %v", code, err)
	}

	return language
}

func partitionsOf(t *testing.T, exponent uint) yacymodel.DHTRingPartitions {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(exponent)
	if err != nil {
		t.Fatal(err)
	}

	return partitions
}
