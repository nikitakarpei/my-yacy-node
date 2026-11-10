package pagerwicontract_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var startedAfter = json.RawMessage(`{"drand":{"chain":"c","round":1,"signature":"s"}}`)

func TestPageRWIStatementRoundTripsEveryFact(t *testing.T) {
	statement := pagerwicontract.PageRWIStatementOf(
		pageRWIRootOf(t),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
		canonicalurltest.CanonicalURLOf(t, "https://example.org/landed"),
		startedAfter,
	)

	read, err := pagerwicontract.UnmarshalPageRWIStatement(
		pagerwicontract.MarshalPageRWIStatement(statement),
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(read, statement) {
		t.Errorf("read statement = %+v, want %+v", read, statement)
	}
}

func TestPageRWIStatementLeavesOutALandedURLThatIsThePageURL(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)
	statement := pagerwicontract.PageRWIStatementOf(
		pageRWIRootOf(t),
		pageURL,
		pageURL,
		startedAfter,
	)

	data := pagerwicontract.MarshalPageRWIStatement(statement)
	read, err := pagerwicontract.UnmarshalPageRWIStatement(data)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "landedURL") {
		t.Errorf("statement %s holds the landed URL", data)
	}
	if read.LandedURL != pageURL {
		t.Errorf("read landed URL = %s, want the page URL %s", read.LandedURL, pageURL)
	}
}

func TestPageRWIStatementSubjectIsThePageRWIRoot(t *testing.T) {
	root := pageRWIRootOf(t)
	data := string(pagerwicontract.MarshalPageRWIStatement(pagerwicontract.PageRWIStatementOf(
		root, canonicalurltest.CanonicalURLOf(t, pageAddress),
		canonicalurltest.CanonicalURLOf(t, pageAddress), startedAfter,
	)))

	for _, part := range []string{
		`"_type":"https://in-toto.io/Statement/v1"`,
		`"predicateType":"` + pagerwicontract.PageRWIPredicateType + `"`,
		`"name":"pageRWIRoot"`,
	} {
		if !strings.Contains(data, part) {
			t.Errorf("statement %s holds no %s", data, part)
		}
	}
}

func TestUnmarshalPageRWIStatementRefusesOtherStatements(t *testing.T) {
	statement := string(pagerwicontract.MarshalPageRWIStatement(pagerwicontract.PageRWIStatementOf(
		pageRWIRootOf(t), canonicalurltest.CanonicalURLOf(t, pageAddress),
		canonicalurltest.CanonicalURLOf(t, pageAddress), startedAfter,
	)))
	for name, data := range map[string]string{
		"no json":       "not a statement",
		"other type":    strings.Replace(statement, "page-rwi-statement", "other", 1),
		"other subject": strings.Replace(statement, `"name":"pageRWIRoot"`, `"name":"body"`, 1),
		"short digest":  strings.Replace(statement, `"sha256":"`, `"sha256":"00`, 1),
		"no hex digest": strings.Replace(statement, `"sha256":"`, `"sha256":"x`, 1),
	} {
		if _, err := pagerwicontract.UnmarshalPageRWIStatement([]byte(data)); err == nil {
			t.Errorf("%s: unmarshal accepted %s", name, data)
		} else if name != "no json" && !errors.Is(err, pagerwicontract.ErrNotAPageRWIStatement) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

func pageRWIRootOf(t *testing.T) yacymodel.PageRWIRoot {
	t.Helper()
	built := pageRWIWithEveryFact(t)
	root, present := yacymodel.PageRWIRootFrom(built.Metadata).Get()
	if !present {
		t.Fatal("no page RWI root")
	}

	return root
}
