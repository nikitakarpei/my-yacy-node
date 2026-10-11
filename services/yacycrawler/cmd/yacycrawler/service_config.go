package main

import (
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	pagevisitlimitsredis "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitlimits/redis"
)

const (
	fetchRetryFloor                          = 500 * time.Millisecond
	fetchRetryCeiling                        = 30 * time.Second
	maxDeferralsPerURL                       = 3
	maxAttemptsPerURL                        = 3
	crawlOrdersAckWait                       = 30 * time.Second
	orderIntakeConcurrency                   = 4
	pendingPageVisitDuplicateWindow          = 2 * time.Minute
	pendingPageVisitAckWaitsPerFetchDeadline = 3

	pageVisitRetention = 30 * 24 * time.Hour

	takenPageVisitRetention = 7 * 24 * time.Hour

	pageVisitLimitRetention = 7 * 24 * time.Hour

	acceptedOrderRetention = 7 * 24 * time.Hour

	crawledPageRetention    = 7 * 24 * time.Hour
	crawledPageBacklogLimit = 5000
)

type ServiceConfig struct {
	CrawlNATSURL            string
	CrawlStateRedis         *redis.Options
	CrawlOrdersSubject      string
	CrawlOrdersDurable      string
	PendingPageVisitDurable string
	ProxyURL                *url.URL
	ProxyDialMode           http.ProxyDialMode
	FetchConcurrency        int
	MaxBodyBytes            int64
	FetchDeadline           time.Duration
	RecrawlGrace            time.Duration
	OpsAddr                 string
	UserAgent               string
}

func (cfg ServiceConfig) SuppressesRecrawl() bool {
	return cfg.RecrawlGrace > 0
}

func (cfg ServiceConfig) PendingPageVisitAckWait() time.Duration {
	return cfg.FetchDeadline * pendingPageVisitAckWaitsPerFetchDeadline
}

func (ServiceConfig) PageVisitRetention() time.Duration {
	return pageVisitRetention
}

func (ServiceConfig) TakenPageVisitRetention() time.Duration {
	return takenPageVisitRetention
}

func (ServiceConfig) PageVisitLimitRetention() time.Duration {
	return pageVisitLimitRetention
}

func (ServiceConfig) AcceptedOrderRetention() time.Duration {
	return acceptedOrderRetention
}

func (ServiceConfig) FetchRetryBounds() retrydelay.Bounds {
	return retrydelay.Bounds{Floor: fetchRetryFloor, Ceiling: fetchRetryCeiling}
}

func (ServiceConfig) MaxPerURL() pagevisitlimitsredis.MaxPerURL {
	return pagevisitlimitsredis.MaxPerURL{
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

func (ServiceConfig) PendingPageVisitDuplicateWindow() time.Duration {
	return pendingPageVisitDuplicateWindow
}
