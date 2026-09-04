package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
)

const BucketName = "YACY_PAGE_VISIT_LIMITS"

func Ensure(
	ctx context.Context,
	js jetstream.JetStream,
	spec jetstreamrecord.BucketSpec,
) error {
	return jetstreamrecord.EnsureBucket(ctx, js, BucketName, spec)
}

type MaxPerURL struct {
	Deferrals int
	Attempts  int
}

type PageVisitLimits struct {
	hostPages  *jetstreamrecord.Records[hostPages]
	pageVisits *jetstreamrecord.Records[pageVisitTaken]
	maxPerURL  MaxPerURL
}

func New(bucket jetstream.KeyValue, maxPerURL MaxPerURL) *PageVisitLimits {
	return &PageVisitLimits{
		hostPages:  jetstreamrecord.New[hostPages](bucket),
		pageVisits: jetstreamrecord.New[pageVisitTaken](bucket),
		maxPerURL:  maxPerURL,
	}
}

func (limits *PageVisitLimits) AdmitsAnotherPageOfHost(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
	host string,
	maxPages int,
) (bool, error) {
	if maxPages == yacycrawlcontract.UnlimitedPagesPerHost {
		return true, nil
	}
	held, err := limits.holdsPageOfHost(ctx, orderID, url)
	if err != nil || held {
		return held, err
	}
	admitted, err := limits.takePageOfHost(ctx, orderID, host, maxPages)
	if err != nil || !admitted {
		return false, err
	}
	return true, limits.holdPageOfHost(ctx, orderID, url)
}

func (limits *PageVisitLimits) holdsPageOfHost(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	taken, err := limits.pageVisits.RecordAt(ctx, pageVisitKeyOf(orderID, url))
	if err != nil {
		return false, fmt.Errorf("read the page of the host held for %s: %w", url, err)
	}
	return taken.HoldsPageOfHost, nil
}

func (limits *PageVisitLimits) takePageOfHost(
	ctx context.Context,
	orderID string,
	host string,
	maxPages int,
) (bool, error) {
	_, taken, err := limits.hostPages.Revise(
		ctx, hostKeyOf(orderID, host), pageWithinHostLimit(maxPages),
	)
	if err != nil {
		return false, fmt.Errorf("take a page of host %s: %w", host, err)
	}
	return taken, nil
}

func pageWithinHostLimit(maxPages int) func(hostPages) (hostPages, bool) {
	return func(pages hostPages) (hostPages, bool) {
		if pages.Pages >= maxPages {
			return pages, false
		}
		pages.Pages++
		return pages, true
	}
}

func (limits *PageVisitLimits) holdPageOfHost(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) error {
	_, _, err := limits.pageVisits.Revise(ctx, pageVisitKeyOf(orderID, url), pageOfHostHeld)
	if err != nil {
		return fmt.Errorf("hold the page of the host for %s: %w", url, err)
	}
	return nil
}

func pageOfHostHeld(taken pageVisitTaken) (pageVisitTaken, bool) {
	if taken.HoldsPageOfHost {
		return taken, false
	}
	taken.HoldsPageOfHost = true
	return taken, true
}

func (limits *PageVisitLimits) AdmitsAnotherDeferral(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	_, deferred, err := limits.pageVisits.Revise(
		ctx, pageVisitKeyOf(orderID, url), limits.anotherDeferral,
	)
	if err != nil {
		return false, fmt.Errorf("take a deferral of the page visit to %s: %w", url, err)
	}
	return deferred, nil
}

func (limits *PageVisitLimits) anotherDeferral(taken pageVisitTaken) (pageVisitTaken, bool) {
	if taken.Deferrals >= limits.maxPerURL.Deferrals {
		return taken, false
	}
	taken.Deferrals++
	return taken, true
}

func (limits *PageVisitLimits) AdmitsAnotherAttempt(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (int, bool, error) {
	standing, attempted, err := limits.pageVisits.Revise(
		ctx, pageVisitKeyOf(orderID, url), limits.anotherAttempt,
	)
	if err != nil {
		return 0, false, fmt.Errorf("take an attempt of the page visit to %s: %w", url, err)
	}
	return standing.Attempts, attempted, nil
}

func (limits *PageVisitLimits) anotherAttempt(taken pageVisitTaken) (pageVisitTaken, bool) {
	if taken.Attempts >= limits.maxPerURL.Attempts {
		return taken, false
	}
	taken.Attempts++
	return taken, true
}

type hostPages struct {
	Pages int `json:"Pages"`
}

type pageVisitTaken struct {
	HoldsPageOfHost bool `json:"HoldsPageOfHost"`
	Deferrals       int  `json:"Deferrals"`
	Attempts        int  `json:"Attempts"`
}

func hostKeyOf(orderID string, host string) string {
	return jetstreamrecord.KeyOf(orderID, "host", host)
}

func pageVisitKeyOf(orderID string, url canonicalurl.CanonicalURL) string {
	return jetstreamrecord.KeyOf(orderID, "page-visit", url.String())
}
