package consolepages

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type hostRow struct {
	Host              string
	PendingPageVisits int64
	visitOutcomes
}

func hostRowFrom(crawlHost yacycrawlcontract.CrawlHost) hostRow {
	return hostRow{
		Host:              crawlHost.Host,
		PendingPageVisits: crawlHost.PendingPageVisits,
		visitOutcomes: visitOutcomesFrom(
			crawlHost.CompletedPageVisitsByResult,
			crawlHost.DroppedPageVisitsByReason,
		),
	}
}
