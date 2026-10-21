package consolepages_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestCrawlDurationShowsItsTwoLargestUnits(t *testing.T) {
	durations := map[time.Duration]string{
		45 * time.Second:              "45s",
		3*time.Minute + 7*time.Second: "3m 7s",
		26*time.Hour + 3*time.Minute:  "1d 2h",
		-time.Minute:                  "0s",
	}
	for duration, text := range durations {
		crawl := finishedCrawlOf(t, "c1")
		crawl.FinishedAt = startedAt.Add(duration)

		answer := consoleOver(
			&fakeCrawler{crawls: map[string]yacycrawlcontract.Crawl{"c1": crawl}},
		).get("/crawls/c1")

		assertShows(
			t,
			answer,
			`<div class="amount">`+text+`</div><div class="muted">Duration</div>`,
		)
	}
}
