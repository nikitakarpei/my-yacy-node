package main

import (
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitlimits"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

const (
	fetchRetryFloor        = 500 * time.Millisecond
	fetchRetryCeiling      = 30 * time.Second
	maxDeferralsPerURL     = 3
	maxDeferral            = time.Hour
	maxAttemptsPerURL      = 3
	crawlOrdersAckWait     = 30 * time.Second
	orderIntakeConcurrency = 4
	fetchDeadlinesPerLease = 3

	pageVisitRetention = 30 * 24 * time.Hour

	pageVisitLimitRetention = 7 * 24 * time.Hour

	crawlRetention = 7 * 24 * time.Hour

	crawledPageRetention    = 7 * 24 * time.Hour
	crawledPageBacklogLimit = 5000
)

type ServiceConfig struct {
	CrawlNATSURL       string
	CrawlStateDir      string
	CrawlStateLimits   pebblevault.MachineLimits
	CrawlOrdersSubject string
	CrawlOrdersDurable string
	ProxyURL           *url.URL
	ProxyDialMode      http.ProxyDialMode
	FetchConcurrency   int
	MaxBodyBytes       int64
	FetchDeadline      time.Duration
	RecrawlGrace       time.Duration
	OpsAddr            string
	APIAddr            string
	UserAgent          string
}

func (cfg ServiceConfig) ServesAPI() bool {
	return cfg.APIAddr != ""
}

func (cfg ServiceConfig) SuppressesRecrawl() bool {
	return cfg.RecrawlGrace > 0
}

func (cfg ServiceConfig) PendingPageVisitLease() time.Duration {
	return cfg.FetchDeadline * fetchDeadlinesPerLease
}

func (ServiceConfig) PageVisitRetention() time.Duration {
	return pageVisitRetention
}

func (ServiceConfig) PageVisitLimitRetention() time.Duration {
	return pageVisitLimitRetention
}

func (ServiceConfig) CrawlRetention() time.Duration {
	return crawlRetention
}

func (ServiceConfig) FetchRetryBounds() retrydelay.Bounds {
	return retrydelay.Bounds{Floor: fetchRetryFloor, Ceiling: fetchRetryCeiling}
}

func (ServiceConfig) MaxDeferral() time.Duration {
	return maxDeferral
}

func (ServiceConfig) MaxPerURL() pagevisitlimits.MaxPerURL {
	return pagevisitlimits.MaxPerURL{
		Deferrals: maxDeferralsPerURL,
		Attempts:  maxAttemptsPerURL,
	}
}

func (ServiceConfig) CrawlOrdersAckWait() time.Duration {
	return crawlOrdersAckWait
}

func (ServiceConfig) OrderIntakeConcurrency() int {
	return orderIntakeConcurrency
}

func (ServiceConfig) CrawledPageRetention() time.Duration {
	return crawledPageRetention
}

func (ServiceConfig) CrawledPageBacklogLimit() uint64 {
	return crawledPageBacklogLimit
}
