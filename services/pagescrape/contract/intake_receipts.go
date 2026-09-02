package pagescrapecontract

import "github.com/nikitakarpei/yacy-rwi-node/canonicalurl"

const (
	IntakeReceiptSubjectPrefix = "page.intake"
	EveryIntakeReceiptSubject  = IntakeReceiptSubjectPrefix + ".>"
)

type CorpusName string

func KeptPageSubjectOf(pageURL canonicalurl.CanonicalURL) string {
	return intakeReceiptSubjectPrefixOf(pageURL) + ".kept"
}

func RejectedPageSubjectOf(pageURL canonicalurl.CanonicalURL) string {
	return intakeReceiptSubjectPrefixOf(pageURL) + ".rejected"
}

func intakeReceiptSubjectPrefixOf(pageURL canonicalurl.CanonicalURL) string {
	return IntakeReceiptSubjectPrefix + "." + pageFingerprintOf(pageURL)
}
