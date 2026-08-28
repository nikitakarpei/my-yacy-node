package main

import (
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
	visitclaimsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/visitclaims/jetstream"
)

const (
	fetchRetryFloor                      = 500 * time.Millisecond
	fetchRetryCeiling                    = 30 * time.Second
	maxDeferralsPerURL                   = 3
	maxAttemptsPerURL                    = 3
	crawlOrdersAckWait                   = 30 * time.Second
	orderIntakeConcurrency               = 4
	pendingVisitDuplicateWindow          = 2 * time.Minute
	pendingVisitAckWaitsPerFetchDeadline = 3

	pageVisitRetention = 30 * 24 * time.Hour
	pageVisitMaxBytes  = 256 << 20

	visitClaimRetention = 7 * 24 * time.Hour
	visitClaimMaxBytes  = 1 << 30

	hostPageAllowanceRetention = 7 * 24 * time.Hour
	hostPageAllowanceMaxBytes  = 256 << 20

	acceptedOrderRetention = 7 * 24 * time.Hour
	acceptedOrderMaxBytes  = 64 << 20
)

type ServiceConfig struct {
	CrawlNATSURL         string
	ScrapeRequestNATSURL string
	CrawlOrdersSubject   string
	CrawlOrdersDurable   string
	PendingVisitDurable  string
	ProxyURL             *url.URL
	ProxyDialMode        http.ProxyDialMode
	FetchConcurrency     int
	MaxBodyBytes         int64
	FetchDeadline        time.Duration
	RecrawlGrace         time.Duration
	OpsAddr              string
	UserAgent            string
}

func (cfg ServiceConfig) SuppressesRecrawl() bool {
	return cfg.RecrawlGrace > 0
}

func (cfg ServiceConfig) PendingVisitAckWait() time.Duration {
	return cfg.FetchDeadline * pendingVisitAckWaitsPerFetchDeadline
}

func (ServiceConfig) PageVisitBucketSpec() jetstreamrecord.BucketSpec {
	return jetstreamrecord.BucketSpec{
		MaxBytes:  pageVisitMaxBytes,
		Retention: pageVisitRetention,
	}
}

func (ServiceConfig) VisitClaimBucketSpec() jetstreamrecord.BucketSpec {
	return jetstreamrecord.BucketSpec{
		MaxBytes:  visitClaimMaxBytes,
		Retention: visitClaimRetention,
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

func (ServiceConfig) VisitClaimLimits() visitclaimsjetstream.ClaimLimits {
	return visitclaimsjetstream.ClaimLimits{
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

func (ServiceConfig) PendingVisitDuplicateWindow() time.Duration {
	return pendingVisitDuplicateWindow
}
