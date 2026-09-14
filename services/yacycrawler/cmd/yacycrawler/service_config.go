package main

import (
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
	pagevisitlimitsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitlimits/jetstream"
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
	pageVisitMaxBytes  = 256 << 20

	takenPageVisitRetention = 7 * 24 * time.Hour
	takenPageVisitMaxBytes  = 1 << 30

	pageVisitLimitRetention = 7 * 24 * time.Hour
	pageVisitLimitMaxBytes  = 256 << 20

	acceptedOrderRetention = 7 * 24 * time.Hour
	acceptedOrderMaxBytes  = 64 << 20

	crawledPageRetention = 7 * 24 * time.Hour
)

type ServiceConfig struct {
	CrawlNATSURL            string
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

func (ServiceConfig) PageVisitBucketSpec() jetstreamrecord.BucketSpec {
	return jetstreamrecord.BucketSpec{
		MaxBytes:  pageVisitMaxBytes,
		Retention: pageVisitRetention,
	}
}

func (ServiceConfig) TakenPageVisitBucketSpec() jetstreamrecord.BucketSpec {
	return jetstreamrecord.BucketSpec{
		MaxBytes:  takenPageVisitMaxBytes,
		Retention: takenPageVisitRetention,
	}
}

func (ServiceConfig) PageVisitLimitBucketSpec() jetstreamrecord.BucketSpec {
	return jetstreamrecord.BucketSpec{
		MaxBytes:  pageVisitLimitMaxBytes,
		Retention: pageVisitLimitRetention,
	}
}

func (ServiceConfig) AcceptedOrderBucketSpec() jetstreamrecord.BucketSpec {
	return jetstreamrecord.BucketSpec{
		MaxBytes:  acceptedOrderMaxBytes,
		Retention: acceptedOrderRetention,
	}
}

func (ServiceConfig) FetchRetryBounds() retrydelay.Bounds {
	return retrydelay.Bounds{Floor: fetchRetryFloor, Ceiling: fetchRetryCeiling}
}

func (ServiceConfig) MaxPerURL() pagevisitlimitsjetstream.MaxPerURL {
	return pagevisitlimitsjetstream.MaxPerURL{
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

func (ServiceConfig) PendingPageVisitDuplicateWindow() time.Duration {
	return pendingPageVisitDuplicateWindow
}
