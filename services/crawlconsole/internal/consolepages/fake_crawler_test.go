package consolepages_test

import (
	"context"
	"errors"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

var errCrawlerDown = errors.New("crawler down")

type crawlsQuery struct {
	states []yacycrawlcontract.CrawlState
	cursor string
}

type fakeCrawler struct {
	down          bool
	hostsDown     bool
	hostPage      yacycrawlcontract.CrawlHostPage
	hostCursors   []string
	status        yacycrawlcontract.CrawlerStatus
	crawls        map[string]yacycrawlcontract.Crawl
	pages         map[string]yacycrawlcontract.CrawlPage
	refusal       string
	placedOrders  []yacycrawlcontract.CrawlOrder
	crawlsQueries []crawlsQuery
	calls         []string
}

func (crawler *fakeCrawler) Place(
	_ context.Context,
	order yacycrawlcontract.CrawlOrder,
) (crawlID string, refusal string, err error) {
	if crawler.down {
		return "", "", errCrawlerDown
	}
	if crawler.refusal != "" {
		return "", crawler.refusal, nil
	}
	crawler.placedOrders = append(crawler.placedOrders, order)
	return crawlIDOf(order), "", nil
}

func crawlIDOf(order yacycrawlcontract.CrawlOrder) string {
	return "crawl-of-" + order.OrderID
}

func (crawler *fakeCrawler) CrawlsIn(
	_ context.Context,
	states []yacycrawlcontract.CrawlState,
	cursor string,
	_ int,
) (yacycrawlcontract.CrawlPage, error) {
	if crawler.down {
		return yacycrawlcontract.CrawlPage{}, errCrawlerDown
	}
	crawler.crawlsQueries = append(
		crawler.crawlsQueries,
		crawlsQuery{states: states, cursor: cursor},
	)
	for state, page := range crawler.pages {
		if slices.Contains(states, yacycrawlcontract.CrawlState(state)) {
			return page, nil
		}
	}
	return yacycrawlcontract.CrawlPage{}, nil
}

func (crawler *fakeCrawler) CrawlOf(
	_ context.Context,
	crawlID string,
) (yacycrawlcontract.Crawl, bool, error) {
	if crawler.down {
		return yacycrawlcontract.Crawl{}, false, errCrawlerDown
	}
	crawl, found := crawler.crawls[crawlID]
	return crawl, found, nil
}

func (crawler *fakeCrawler) HostsOf(
	_ context.Context,
	_ string,
	cursor string,
	_ int,
) (yacycrawlcontract.CrawlHostPage, error) {
	if crawler.down || crawler.hostsDown {
		return yacycrawlcontract.CrawlHostPage{}, errCrawlerDown
	}
	crawler.hostCursors = append(crawler.hostCursors, cursor)
	return crawler.hostPage, nil
}

func (crawler *fakeCrawler) Cancel(_ context.Context, crawlID string) error {
	return crawler.called("cancel " + crawlID)
}

func (crawler *fakeCrawler) Pause(_ context.Context, crawlID string) error {
	return crawler.called("pause " + crawlID)
}

func (crawler *fakeCrawler) Resume(_ context.Context, crawlID string) error {
	return crawler.called("resume " + crawlID)
}

func (crawler *fakeCrawler) CrawlerStatus(
	context.Context,
) (yacycrawlcontract.CrawlerStatus, error) {
	if crawler.down {
		return yacycrawlcontract.CrawlerStatus{}, errCrawlerDown
	}
	return crawler.status, nil
}

func (crawler *fakeCrawler) PauseCrawler(context.Context) error {
	return crawler.called("pause crawler")
}

func (crawler *fakeCrawler) ResumeCrawler(context.Context) error {
	return crawler.called("resume crawler")
}

func (crawler *fakeCrawler) called(call string) error {
	if crawler.down {
		return errCrawlerDown
	}
	crawler.calls = append(crawler.calls, call)
	return nil
}

type recordingObserver struct {
	failedCalls     int
	unrenderedPages []string
}

func (observer *recordingObserver) CrawlerCallFailed(context.Context, error) {
	observer.failedCalls++
}

func (observer *recordingObserver) PageRenderFailed(_ context.Context, screen string, _ error) {
	observer.unrenderedPages = append(observer.unrenderedPages, screen)
}
