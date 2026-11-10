package pagescrapecontract_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

var (
	startedAfter = json.RawMessage(`{"drand":{"chain":"c","round":1,"signature":"s"}}`)
	offerBody    = []byte(`{"PageURL":"https://example.org/a"}`)
)

func TestPageOfferStatementCoversTheOfferBodyItWasMadeOf(t *testing.T) {
	statement := pagescrapecontract.PageOfferStatementOf(offerBody, fetchURLOf(t), startedAfter)

	if !statement.Covers(offerBody) {
		t.Error("the statement does not cover the offer body it was made of")
	}
	if statement.Covers(append([]byte(" "), offerBody...)) {
		t.Error("the statement covers another offer body")
	}
}

func TestPageOfferStatementRoundTripsEveryFact(t *testing.T) {
	statement := pagescrapecontract.PageOfferStatementOf(offerBody, fetchURLOf(t), startedAfter)

	read, err := pagescrapecontract.UnmarshalPageOfferStatement(
		pagescrapecontract.MarshalPageOfferStatement(statement),
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(read, statement) {
		t.Errorf("read statement = %+v, want %+v", read, statement)
	}
}

func TestPageOfferStatementIsAnInTotoStatementOfItsPredicateType(t *testing.T) {
	data := string(pagescrapecontract.MarshalPageOfferStatement(
		pagescrapecontract.PageOfferStatementOf(offerBody, fetchURLOf(t), startedAfter),
	))

	for _, part := range []string{
		`"_type":"https://in-toto.io/Statement/v1"`,
		`"predicateType":"` + pagescrapecontract.PageOfferPredicateType + `"`,
		`"name":"pageOffer"`,
	} {
		if !strings.Contains(data, part) {
			t.Errorf("statement %s holds no %s", data, part)
		}
	}
}

func TestUnmarshalPageOfferStatementRefusesOtherStatements(t *testing.T) {
	statement := string(pagescrapecontract.MarshalPageOfferStatement(
		pagescrapecontract.PageOfferStatementOf(offerBody, fetchURLOf(t), startedAfter),
	))
	for name, data := range map[string]string{
		"no json":        "not a statement",
		"other type":     strings.Replace(statement, "page-offer-statement", "other", 1),
		"other subject":  strings.Replace(statement, `"name":"pageOffer"`, `"name":"body"`, 1),
		"short digest":   strings.Replace(statement, `"sha256":"`, `"sha256":"00`, 1),
		"other envelope": strings.Replace(statement, "Statement/v1", "Statement/v9", 1),
	} {
		if _, err := pagescrapecontract.UnmarshalPageOfferStatement([]byte(data)); err == nil {
			t.Errorf("%s: unmarshal accepted %s", name, data)
		} else if name != "no json" && !errors.Is(err, pagescrapecontract.ErrNotAPageOfferStatement) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

func fetchURLOf(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()

	return canonicalurltest.CanonicalURLOf(t, "https://example.org/a")
}
