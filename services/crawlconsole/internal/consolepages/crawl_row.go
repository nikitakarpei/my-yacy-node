package consolepages

import (
	"cmp"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type crawlRow struct {
	CrawlID               string
	Name                  string
	Label                 string
	FirstSeedURL          string
	AmountOfOtherSeedURLs int64
	State                 yacycrawlcontract.CrawlState
	PendingPageVisits     int64
	Outcomes              visitOutcomes
	UnadmittedLinks       int64
	StartedAt             time.Time
	FinishedAt            time.Time
	Duration              crawlDuration
}

func crawlRowFrom(crawl yacycrawlcontract.Crawl, now time.Time) crawlRow {
	firstSeedURL := crawl.FirstSeedURL.String()
	return crawlRow{
		CrawlID:               crawl.CrawlID,
		Name:                  cmp.Or(crawl.Label, firstSeedURL),
		Label:                 crawl.Label,
		FirstSeedURL:          firstSeedURL,
		AmountOfOtherSeedURLs: max(crawl.AmountOfSeedURLs-1, 0),
		State:                 crawl.State,
		PendingPageVisits:     crawl.PendingPageVisits,
		Outcomes: visitOutcomesFrom(
			crawl.CompletedPageVisitsByResult,
			crawl.DroppedPageVisitsByReason,
		),
		UnadmittedLinks: crawl.UnadmittedLinks,
		StartedAt:       crawl.StartedAt,
		FinishedAt:      crawl.FinishedAt,
		Duration:        crawlDurationOf(crawl.StartedAt, crawl.FinishedAt, now),
	}
}

func (row crawlRow) IsFinished() bool {
	return !row.FinishedAt.IsZero()
}

func (row crawlRow) IsPausable() bool {
	return yacycrawlcontract.CrawlPause.Changes(row.State)
}

func (row crawlRow) IsResumable() bool {
	return yacycrawlcontract.CrawlResumption.Changes(row.State)
}

func (row crawlRow) IsCancellable() bool {
	return yacycrawlcontract.CrawlCancellation.Changes(row.State)
}
