package pagescrapecontract

import "github.com/nikitakarpei/yacy-rwi-node/canonicalurl"

const (
	ScrapeRequestsStreamName    = "SCRAPE_REQUESTS"
	ScrapeRequestSubject        = "scrape.request"
	ScrapeScheduleSubjectPrefix = "scrape.schedule"
	EveryScrapeScheduleSubject  = ScrapeScheduleSubjectPrefix + ".*"
)

func ScrapeScheduleSubjectOf(pageURL canonicalurl.CanonicalURL) string {
	return ScrapeScheduleSubjectPrefix + "." + pageFingerprintOf(pageURL)
}
