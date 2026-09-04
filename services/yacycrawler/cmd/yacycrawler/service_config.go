package main

import (
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
	pagevisitclaimsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitclaims/jetstream"
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

	pageVisitClaimRetention = 7 * 24 * time.Hour
	pageVisitClaimMaxBytes  = 1 << 30

	hostPageAllowanceRetention = 7 * 24 * time.Hour
	hostPageAllowanceMaxBytes  = 256 << 20

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

func (ServiceConfig) PageVisitClaimBucketSpec() jetstreamrecord.BucketSpec {
	return jetstreamrecord.BucketSpec{
		MaxBytes:  pageVisitClaimMaxBytes,
		Retention: pageVisitClaimRetention,
	}
}

func (ServiceConfig) HostPageAllowanceBucketSpec() jetstreamrecord.BucketSpec {
	return jetstreamrecord.BucketSpec{
		MaxBytes:  hostPageAllowanceMaxBytes,
		Retention: hostPageAllowanceRetention,
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

func (ServiceConfig) PageVisitClaimLimits() pagevisitclaimsjetstream.ClaimLimits {
	return pagevisitclaimsjetstream.ClaimLimits{
		MaxDeferralsPerURL: maxDeferralsPerURL,
		MaxAttemptsPerURL:  maxAttemptsPerURL,
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
