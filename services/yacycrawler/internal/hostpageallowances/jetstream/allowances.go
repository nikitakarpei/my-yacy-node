package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
)

const BucketName = "YACY_HOST_PAGE_ALLOWANCES"

func Ensure(
	ctx context.Context,
	js jetstream.JetStream,
	spec jetstreamrecord.BucketSpec,
) error {
	return jetstreamrecord.EnsureBucket(ctx, js, BucketName, spec)
}

type Allowances struct {
	hostPages       *jetstreamrecord.Records[hostPages]
	spentPageVisits *jetstreamrecord.Records[spentPageVisit]
}

func New(bucket jetstream.KeyValue) *Allowances {
	return &Allowances{
		hostPages:       jetstreamrecord.New[hostPages](bucket),
		spentPageVisits: jetstreamrecord.New[spentPageVisit](bucket),
	}
}

func (a *Allowances) HoldsHostPage(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
	host string,
	maxPages int,
) (bool, error) {
	if maxPages == yacycrawlcontract.UnlimitedPagesPerHost {
		return true, nil
	}
	alreadySpent, err := a.pageSpentOnPageVisit(ctx, orderID, url)
	if err != nil {
		return false, err
	}
	if alreadySpent {
		return true, nil
	}
	spent, err := a.takePageOfHost(ctx, orderID, host, maxPages)
	if err != nil || !spent {
		return false, err
	}
	if err := a.markPageSpentOnPageVisit(ctx, orderID, url); err != nil {
		return false, err
	}
	return true, nil
}

func (a *Allowances) pageSpentOnPageVisit(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	pageVisit, err := a.spentPageVisits.RecordAt(ctx, pageVisitKeyOf(orderID, url))
	if err != nil {
		return false, fmt.Errorf("read the page spent on the page pageVisit to %s: %w", url, err)
	}
	return pageVisit.PageSpent, nil
}

func (a *Allowances) takePageOfHost(
	ctx context.Context,
	orderID string,
	host string,
	maxPages int,
) (bool, error) {
	_, taken, err := a.hostPages.Revise(
		ctx, hostKeyOf(orderID, host), pageWithinAllowance(maxPages),
	)
	if err != nil {
		return false, fmt.Errorf("spend a page of host %s: %w", host, err)
	}
	return taken, nil
}

func pageWithinAllowance(maxPages int) func(hostPages) (hostPages, bool) {
	return func(pages hostPages) (hostPages, bool) {
		if pages.Pages >= maxPages {
			return pages, false
		}
		pages.Pages++
		return pages, true
	}
}

func (a *Allowances) markPageSpentOnPageVisit(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) error {
	_, _, err := a.spentPageVisits.Revise(ctx, pageVisitKeyOf(orderID, url), markPageSpent)
	if err != nil {
		return fmt.Errorf("mark the page spent on the page pageVisit to %s: %w", url, err)
	}
	return nil
}

func markPageSpent(pageVisit spentPageVisit) (spentPageVisit, bool) {
	if pageVisit.PageSpent {
		return pageVisit, false
	}
	pageVisit.PageSpent = true
	return pageVisit, true
}

type hostPages struct {
	Pages int `json:"Pages"`
}

type spentPageVisit struct {
	PageSpent bool `json:"PageSpent"`
}

func hostKeyOf(orderID string, host string) string {
	return jetstreamrecord.KeyOf(orderID, "host", host)
}

func pageVisitKeyOf(orderID string, url canonicalurl.CanonicalURL) string {
	return jetstreamrecord.KeyOf(orderID, "pageVisit", url.String())
}
