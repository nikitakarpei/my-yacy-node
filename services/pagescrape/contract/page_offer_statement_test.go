package pagescrapecontract_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

var startedAfter = json.RawMessage(`{"drand":{"chain":"c","round":1,"signature":"s"}}`)

func TestPageOfferStatementOfStatesTheOfferedPage(t *testing.T) {
	page := offeredPageWithEveryFact(t)
	fetchURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")

	statement := pagescrapecontract.PageOfferStatementOf(page, fetchURL, startedAfter)

	if statement.PageURL != page.PageURL || statement.LandedURL != page.LandedURL ||
		statement.FetchURL != fetchURL || statement.ContentType != page.ContentType ||
		!statement.PageModifiedAt.Equal(page.PageModifiedAt) ||
		string(statement.StartedAfter) != string(startedAfter) {
		t.Errorf("statement = %+v, for page %+v", statement, page)
	}
	if !reflect.DeepEqual(
		statement,
		pagescrapecontract.PageOfferStatementOf(page, fetchURL, startedAfter),
	) {
		t.Error("one page gives two statements")
	}
}

func TestPageOfferStatementCommitsToTheBody(t *testing.T) {
	page := offeredPageWithEveryFact(t)
	changed := offeredPageWithEveryFact(t)
	changed.Body = append(changed.Body, '!')
	fetchURL := page.PageURL

	if pagescrapecontract.PageOfferStatementOf(page, fetchURL, startedAfter).BodyDigest ==
		pagescrapecontract.PageOfferStatementOf(changed, fetchURL, startedAfter).BodyDigest {
		t.Error("two bodies give one body digest")
	}
}

func TestPageOfferStatementRoundTripsEveryFact(t *testing.T) {
	page := offeredPageWithEveryFact(t)
	statement := pagescrapecontract.PageOfferStatementOf(page, page.PageURL, startedAfter)

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
	page := offeredPageWithEveryFact(t)
	data := pagescrapecontract.MarshalPageOfferStatement(
		pagescrapecontract.PageOfferStatementOf(page, page.PageURL, startedAfter),
	)

	for _, part := range []string{
		`"_type":"https://in-toto.io/Statement/v1"`,
		`"predicateType":"` + pagescrapecontract.PageOfferPredicateType + `"`,
		`"name":"body"`,
	} {
		if !strings.Contains(string(data), part) {
			t.Errorf("statement %s holds no %s", data, part)
		}
	}
}

func TestUnmarshalPageOfferStatementRefusesOtherStatements(t *testing.T) {
	page := offeredPageWithEveryFact(t)
	statement := string(pagescrapecontract.MarshalPageOfferStatement(
		pagescrapecontract.PageOfferStatementOf(page, page.PageURL, startedAfter),
	))
	for name, data := range map[string]string{
		"no json":        "not a statement",
		"other type":     strings.Replace(statement, "page-offer-statement", "other", 1),
		"other subject":  strings.Replace(statement, `"name":"body"`, `"name":"other"`, 1),
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

func offeredPageWithEveryFact(t *testing.T) pagescrapecontract.OfferedPage {
	t.Helper()

	return pagescrapecontract.OfferedPage{
		PageURL:         canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		LandedURL:       canonicalurltest.CanonicalURLOf(t, "https://example.org/b"),
		ContentType:     "text/html",
		Body:            []byte("<html>hi</html>"),
		RobotsTagValues: []string{"noarchive"},
		PageModifiedAt:  time.Date(2026, time.October, 30, 1, 2, 3, 0, time.UTC),
		FetchedAt:       fetchedAt,
	}
}
