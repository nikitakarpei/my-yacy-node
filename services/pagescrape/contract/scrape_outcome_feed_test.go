package pagescrapecontract_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestEveryOutcomeSubjectIsUnderTheScrapeOutcomeSubjectsOfThePage(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")

	subjects := pagescrapecontract.ScrapeOutcomeSubjectsOf(pageURL)
	outcomes := map[string]string{
		"kept":     pagescrapecontract.KeptPageOutcomeSubjectOf(pageURL),
		"rejected": pagescrapecontract.RejectedPageOutcomeSubjectOf(pageURL),
		"failed":   pagescrapecontract.ScrapeFailureOutcomeSubjectOf(pageURL),
	}

	if !strings.HasSuffix(subjects, ".*") {
		t.Errorf("outcome subjects %q are not a wildcard", subjects)
	}
	under := strings.TrimSuffix(subjects, "*")
	taken := make(map[string]string, len(outcomes))
	for outcome, subject := range outcomes {
		if !strings.HasPrefix(subject, under) {
			t.Errorf("the %s subject %q is not under %q", outcome, subject, subjects)
		}
		if shared, taken := taken[subject]; taken {
			t.Errorf("%s and %s share the subject %q", outcome, shared, subject)
		}
		taken[subject] = outcome
	}
}

func TestScrapeOutcomeSubjectsAreUniquePerPage(t *testing.T) {
	subjects := pagescrapecontract.ScrapeOutcomeSubjectsOf(
		canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
	)
	other := pagescrapecontract.ScrapeOutcomeSubjectsOf(
		canonicalurltest.CanonicalURLOf(t, "https://example.org/b"),
	)

	if subjects == other {
		t.Errorf("two pages share the outcome subjects %q", subjects)
	}
}

func TestScrapeOutcomeSubjectFromAnIntakeReceiptSubjectKeepsThePageAndTheOutcome(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")

	receipts := map[string]string{
		pagescrapecontract.KeptPageSubjectOf(pageURL): pagescrapecontract.KeptPageOutcomeSubjectOf(
			pageURL,
		),
		pagescrapecontract.RejectedPageSubjectOf(
			pageURL,
		): pagescrapecontract.RejectedPageOutcomeSubjectOf(pageURL),
	}
	for receiptSubject, want := range receipts {
		got, err := pagescrapecontract.ScrapeOutcomeSubjectFrom(receiptSubject)
		if err != nil {
			t.Fatalf("outcome subject from %q: %v", receiptSubject, err)
		}
		if got != want {
			t.Errorf("outcome subject from %q = %q, want %q", receiptSubject, got, want)
		}
	}
}

func TestScrapeOutcomeSubjectFromASubjectThatIsNoReceipt(t *testing.T) {
	for _, subject := range []string{"", "page.intake.fingerprint", "scrape.request", "a.b.c.d"} {
		if _, err := pagescrapecontract.ScrapeOutcomeSubjectFrom(subject); err == nil {
			t.Errorf("want an error for the subject %q", subject)
		}
	}
}
