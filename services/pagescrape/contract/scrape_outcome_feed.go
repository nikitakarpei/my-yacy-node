package pagescrapecontract

import (
	"fmt"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const ScrapeOutcomeSubjectPrefix = "scrape.outcome"

func ScrapeOutcomeSubjectsOf(pageURL canonicalurl.CanonicalURL) string {
	return scrapeOutcomeSubjectPrefixOf(pageURL) + ".*"
}

func KeptPageOutcomeSubjectOf(pageURL canonicalurl.CanonicalURL) string {
	return scrapeOutcomeSubjectPrefixOf(pageURL) + ".kept"
}

func RejectedPageOutcomeSubjectOf(pageURL canonicalurl.CanonicalURL) string {
	return scrapeOutcomeSubjectPrefixOf(pageURL) + ".rejected"
}

func ScrapeFailureOutcomeSubjectOf(pageURL canonicalurl.CanonicalURL) string {
	return scrapeOutcomeSubjectPrefixOf(pageURL) + ".failed"
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
