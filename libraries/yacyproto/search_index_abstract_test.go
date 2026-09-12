package yacyproto_test

import (
	"context"
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

	response, err := yacyproto.ParseSearchResponse(context.Background(), yacyproto.Message{
		"indexabstract." + queryWord.String(): wireForm,
	})
	if err != nil {
		t.Fatalf("ParseSearchResponse: %v", err)
	}

	return response.IndexAbstract[queryWord]
}

func TestASearchResponseCarriesTheDocumentsHeldForAQueryWord(t *testing.T) {
	t.Parallel()

	queryWord := sampleHash(t, "alpha")
	documents := []yacymodel.URLHash{
		mustParseURLHash(t, "bbbbbbAAAAAA"),
		mustParseURLHash(t, "ccccccAAAAAA"),
		mustParseURLHash(t, "aaaaaaBBBBBB"),
	}

	message := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash][]yacymodel.URLHash{queryWord: documents},
	}.Encode()
	response, err := yacyproto.ParseSearchResponse(context.Background(), message)
	if err != nil {
		t.Fatalf("ParseSearchResponse: %v", err)
	}

	if !slices.Equal(response.IndexAbstract[queryWord], documents) {
		t.Fatalf(
			"documents = %v, want %v", response.IndexAbstract[queryWord], documents,
		)
	}
}

func TestASearchResponseGroupsTheDocumentsItCarriesByHost(t *testing.T) {
	t.Parallel()

	queryWord := sampleHash(t, "alpha")
	message := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash][]yacymodel.URLHash{queryWord: {
			mustParseURLHash(t, "bbbbbbAAAAAA"),
			mustParseURLHash(t, "aaaaaaBBBBBB"),
			mustParseURLHash(t, "ccccccAAAAAA"),
		}},
	}.Encode()

	want := "{AAAAAA:bbbbbbcccccc,BBBBBB:aaaaaa}"
	if got := message["indexabstract."+queryWord.String()]; got != want {
		t.Fatalf("index abstract = %q, want %q", got, want)
	}
}

func TestASearchResponseCarriesNoDocumentForAQueryWordItHoldsNoneFor(t *testing.T) {
	t.Parallel()

	queryWord := sampleHash(t, "alpha")
	message := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash][]yacymodel.URLHash{queryWord: nil},
	}.Encode()

	if got := message["indexabstract."+queryWord.String()]; got != "{}" {
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
