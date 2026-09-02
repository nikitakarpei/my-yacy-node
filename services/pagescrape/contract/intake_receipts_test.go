package pagescrapecontract_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestKeptPageRoundTrip(t *testing.T) {
	kept := pagescrapecontract.KeptPage{
		PageURL: canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		Corpus:  "corpusmarkdown",
	}

	data, err := pagescrapecontract.MarshalKeptPage(kept)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := pagescrapecontract.UnmarshalKeptPage(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != kept {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", kept, got)
	}
}

func TestUnmarshalKeptPageWithoutACorpus(t *testing.T) {
	if _, err := pagescrapecontract.UnmarshalKeptPage(
		[]byte(`{"PageURL":"https://example.org/a"}`),
	); err == nil {
		t.Fatal("want an error for a kept page that names no corpus")
	}
}

func TestUnmarshalKeptPageInvalidJSON(t *testing.T) {
	if _, err := pagescrapecontract.UnmarshalKeptPage([]byte("not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}

func TestRejectedPageRoundTrip(t *testing.T) {
	rejected := pagescrapecontract.RejectedPage{
		PageURL: canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		Corpus:  "corpustext",
	}

	data, err := pagescrapecontract.MarshalRejectedPage(rejected)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := pagescrapecontract.UnmarshalRejectedPage(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != rejected {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", rejected, got)
	}
}

func TestUnmarshalRejectedPageWithoutACorpus(t *testing.T) {
	if _, err := pagescrapecontract.UnmarshalRejectedPage(
		[]byte(`{"PageURL":"https://example.org/a"}`),
	); err == nil {
		t.Fatal("want an error for a rejected page that names no corpus")
	}
}

func TestUnmarshalRejectedPageInvalidJSON(t *testing.T) {
	if _, err := pagescrapecontract.UnmarshalRejectedPage([]byte("not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}

func TestIntakeReceiptSubjectsSeparateTheOutcomeAndThePage(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")

	kept := pagescrapecontract.KeptPageSubjectOf(pageURL)
	rejected := pagescrapecontract.RejectedPageSubjectOf(pageURL)
	otherPage := pagescrapecontract.KeptPageSubjectOf(
		canonicalurltest.CanonicalURLOf(t, "https://example.org/b"),
	)

	if !strings.HasPrefix(kept, pagescrapecontract.IntakeReceiptSubjectPrefix+".") {
		t.Errorf("subject %q does not start with the receipt prefix", kept)
	}
	if kept == rejected {
		t.Errorf("both outcomes share the subject %q", kept)
	}
	if kept == otherPage {
		t.Errorf("two pages share the subject %q", kept)
	}
}
