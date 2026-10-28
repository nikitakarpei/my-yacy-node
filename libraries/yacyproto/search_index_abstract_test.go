package yacyproto_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func mustParseURLHash(t *testing.T, raw string) yacymodel.URLHash {
	t.Helper()
	hash, err := yacymodel.ParseURLHash(raw)
	if err != nil {
		t.Fatal(err)
	}

	return hash
}

func indexAbstractOfQueryWord(
	t *testing.T,
	queryWord yacymodel.Hash,
	wireForm string,
) []yacymodel.URLHash {
	t.Helper()

	response, err := yacyproto.ParseSearchResponse(context.Background(), yacyproto.ResponseFields{
		"indexabstract." + queryWord.String(): wireForm,
	})
	if err != nil {
		t.Fatalf("ParseSearchResponse: %v", err)
	}

	return response.IndexAbstract[queryWord].InHashOrder()
}

func TestASearchResponseCarriesTheDocumentsHeldForAQueryWord(t *testing.T) {
	t.Parallel()

	queryWord := sampleHash(t, "alpha")
	documents := []yacymodel.URLHash{
		mustParseURLHash(t, "bbbbbbAAAAAA"),
		mustParseURLHash(t, "ccccccAAAAAA"),
		mustParseURLHash(t, "aaaaaaBBBBBB"),
	}

	fields := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash]yacymodel.URLHashes{
			queryWord: yacymodel.URLHashesOf(documents...),
		},
	}.Encode()
	response, err := yacyproto.ParseSearchResponse(context.Background(), fields)
	if err != nil {
		t.Fatalf("ParseSearchResponse: %v", err)
	}

	if got := response.IndexAbstract[queryWord]; !reflect.DeepEqual(
		got, yacymodel.URLHashesOf(documents...),
	) {
		t.Fatalf("documents = %v, want %v", got.InHashOrder(), documents)
	}
}

func TestASearchResponseGroupsTheDocumentsItCarriesByHost(t *testing.T) {
	t.Parallel()

	queryWord := sampleHash(t, "alpha")
	fields := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash]yacymodel.URLHashes{queryWord: yacymodel.URLHashesOf(
			mustParseURLHash(t, "bbbbbbAAAAAA"),
			mustParseURLHash(t, "aaaaaaBBBBBB"),
			mustParseURLHash(t, "ccccccAAAAAA"),
		)},
	}.Encode()

	want := "{AAAAAA:bbbbbbcccccc,BBBBBB:aaaaaa}"
	if got := fields["indexabstract."+queryWord.String()]; got != want {
		t.Fatalf("index abstract = %q, want %q", got, want)
	}
}

func TestASearchResponseCarriesNoDocumentForAQueryWordItHoldsNoneFor(t *testing.T) {
	t.Parallel()

	queryWord := sampleHash(t, "alpha")
	fields := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash]yacymodel.URLHashes{queryWord: {}},
	}.Encode()

	if got := fields["indexabstract."+queryWord.String()]; got != "{}" {
		t.Fatalf("index abstract = %q, want {}", got)
	}
	if got := indexAbstractOfQueryWord(t, queryWord, "{}"); len(got) != 0 {
		t.Fatalf("documents = %v, want none", got)
	}
}

func TestASearchResponseSkipsAGroupOfDocumentsItCannotRead(t *testing.T) {
	t.Parallel()

	queryWord := sampleHash(t, "alpha")
	got := indexAbstractOfQueryWord(t, queryWord, "{AAAAAA:bbbbbb,short:aaaaaa,BBBBBB:}")
	if !slices.Equal(got, []yacymodel.URLHash{mustParseURLHash(t, "bbbbbbAAAAAA")}) {
		t.Fatalf("documents = %v, want the one readable document", got)
	}
}

func TestASearchResponseCarriesNoDocumentOfAnIndexAbstractWithoutBraces(t *testing.T) {
	t.Parallel()

	queryWord := sampleHash(t, "alpha")
	if got := indexAbstractOfQueryWord(t, queryWord, "AAAAAA:bbbbbb"); len(got) != 0 {
		t.Fatalf("documents = %v, want none", got)
	}
}
