package pagescrapecontract

import (
	"fmt"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const ScrapeOutcomeSubjectPrefix = "scrape.outcome"

type ScrapeOutcome string

const (
	PageKept     ScrapeOutcome = "kept"
	PageRejected ScrapeOutcome = "rejected"
	ScrapeFailed ScrapeOutcome = "failed"
)

func ScrapeOutcomeSubjectsOf(pageURL canonicalurl.CanonicalURL) string {
	return scrapeOutcomeSubjectPrefixOf(pageURL) + ".*"
}

func KeptPageOutcomeSubjectOf(pageURL canonicalurl.CanonicalURL) string {
	return scrapeOutcomeSubjectOf(pageURL, PageKept)
}

func RejectedPageOutcomeSubjectOf(pageURL canonicalurl.CanonicalURL) string {
	return scrapeOutcomeSubjectOf(pageURL, PageRejected)
}

func ScrapeFailureOutcomeSubjectOf(pageURL canonicalurl.CanonicalURL) string {
	return scrapeOutcomeSubjectOf(pageURL, ScrapeFailed)
}

func scrapeOutcomeSubjectOf(
	pageURL canonicalurl.CanonicalURL,
	outcome ScrapeOutcome,
) string {
	return scrapeOutcomeSubjectPrefixOf(pageURL) + "." + string(outcome)
}

func ScrapeOutcomeOn(feedSubject string) (ScrapeOutcome, error) {
	const feedTokenCount = 4
	tokens := strings.Split(feedSubject, ".")
	if len(tokens) != feedTokenCount ||
		tokens[0]+"."+tokens[1] != ScrapeOutcomeSubjectPrefix {
		return "", fmt.Errorf("%q is no scrape outcome subject", feedSubject)
	}
	switch outcome := ScrapeOutcome(tokens[3]); outcome {
	case PageKept, PageRejected, ScrapeFailed:
		return outcome, nil
	default:
		return "", fmt.Errorf("%q carries no known scrape outcome", feedSubject)
	}
}

func scrapeOutcomeSubjectPrefixOf(pageURL canonicalurl.CanonicalURL) string {
	return ScrapeOutcomeSubjectPrefix + "." + pageFingerprintOf(pageURL)
}

func ScrapeOutcomeSubjectFrom(intakeReceiptSubject string) (string, error) {
	const receiptTokenCount = 4
	tokens := strings.Split(intakeReceiptSubject, ".")
	if len(tokens) != receiptTokenCount ||
		tokens[0]+"."+tokens[1] != IntakeReceiptSubjectPrefix {
		return "", fmt.Errorf("%q is no intake receipt subject", intakeReceiptSubject)
	}
	pageFingerprint, outcome := tokens[2], tokens[3]
	return ScrapeOutcomeSubjectPrefix + "." + pageFingerprint + "." + outcome, nil
}
