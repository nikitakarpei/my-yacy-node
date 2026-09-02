package pagescrapecontract

import "github.com/nikitakarpei/yacy-rwi-node/canonicalurl"

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
