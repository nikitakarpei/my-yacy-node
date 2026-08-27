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
	hostPages   *jetstreamrecord.Records[hostPages]
	spentVisits *jetstreamrecord.Records[spentVisit]
}

func New(bucket jetstream.KeyValue) *Allowances {
	return &Allowances{
		hostPages:   jetstreamrecord.New[hostPages](bucket),
		spentVisits: jetstreamrecord.New[spentVisit](bucket),
	}
}

func (a *Allowances) SpendPage(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
	host string,
	maxPages int,
) (bool, error) {
	if maxPages == yacycrawlcontract.UnlimitedPagesPerHost {
		return true, nil
	}
	alreadySpent, err := a.pageSpentOnVisit(ctx, orderID, url)
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
	if err := a.markPageSpentOnVisit(ctx, orderID, url); err != nil {
		return false, err
	}
	return true, nil
}

func (a *Allowances) pageSpentOnVisit(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	visit, err := a.spentVisits.RecordAt(ctx, visitKeyOf(orderID, url))
	if err != nil {
		return false, fmt.Errorf("read the page spent on the visit to %s: %w", url, err)
	}
	return visit.PageSpent, nil
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

func (a *Allowances) markPageSpentOnVisit(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) error {
	_, _, err := a.spentVisits.Revise(ctx, visitKeyOf(orderID, url), markPageSpent)
	if err != nil {
		return fmt.Errorf("mark the page spent on the visit to %s: %w", url, err)
	}
	return nil
}

func markPageSpent(visit spentVisit) (spentVisit, bool) {
	if visit.PageSpent {
		return visit, false
	}
	visit.PageSpent = true
	return visit, true
}

type hostPages struct {
	Pages int `json:"Pages"`
}

type spentVisit struct {
	PageSpent bool `json:"PageSpent"`
}

func hostKeyOf(orderID string, host string) string {
	return jetstreamrecord.KeyOf(orderID, "host", host)
}

func visitKeyOf(orderID string, url canonicalurl.CanonicalURL) string {
	return jetstreamrecord.KeyOf(orderID, "visit", url.String())
}
