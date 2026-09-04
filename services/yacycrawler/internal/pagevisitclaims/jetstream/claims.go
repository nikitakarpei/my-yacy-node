package jetstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitclaim"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
)

const BucketName = "YACY_VISIT_CLAIMS"

var errUnclaimedPageVisit = errors.New("no holder claimed the page visit")

func Ensure(
	ctx context.Context,
	js jetstream.JetStream,
	spec jetstreamrecord.BucketSpec,
) error {
	return jetstreamrecord.EnsureBucket(ctx, js, BucketName, spec)
}

type ClaimLimits struct {
	MaxDeferralsPerURL int
	MaxAttemptsPerURL  int
}

type Claims struct {
	pageVisitClaims *jetstreamrecord.Records[pageVisitClaim]
	limits          ClaimLimits
}

func New(bucket jetstream.KeyValue, limits ClaimLimits) *Claims {
	return &Claims{
		pageVisitClaims: jetstreamrecord.New[pageVisitClaim](bucket),
		limits:          limits,
	}
}

func (c *Claims) Claim(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
	holder string,
) (pagevisitclaim.Claim, error) {
	standing, taken, err := c.pageVisitClaims.Revise(
		ctx, pageVisitKeyOf(orderID, url), takenBy(holder),
	)
	if err != nil {
		return pagevisitclaim.Unanswered, fmt.Errorf("claim the page visit to %s: %w", url, err)
	}
	if taken {
		return pagevisitclaim.Taken, nil
	}
	if standing.Holder == holder {
		return pagevisitclaim.Resumed, nil
	}
	return pagevisitclaim.HeldElsewhere, nil
}

func takenBy(holder string) func(pageVisitClaim) (pageVisitClaim, bool) {
	return func(claim pageVisitClaim) (pageVisitClaim, bool) {
		if claim.Holder != "" {
			return claim, false
		}
		claim.Holder = holder
		return claim, true
	}
}

func (c *Claims) Defer(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	standing, deferred, err := c.pageVisitClaims.Revise(
		ctx, pageVisitKeyOf(orderID, url), c.spendDeferral,
	)
	if err != nil {
		return false, fmt.Errorf("defer the page visit to %s: %w", url, err)
	}
	if standing.Holder == "" {
		return false, fmt.Errorf("defer the page visit to %s: %w", url, errUnclaimedPageVisit)
	}
	return deferred, nil
}

func (c *Claims) spendDeferral(claim pageVisitClaim) (pageVisitClaim, bool) {
	if claim.Holder == "" || claim.Deferrals >= c.limits.MaxDeferralsPerURL {
		return claim, false
	}
	claim.Deferrals++
	return claim, true
}

func (c *Claims) Retry(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (int, bool, error) {
	standing, retried, err := c.pageVisitClaims.Revise(
		ctx, pageVisitKeyOf(orderID, url), c.spendAttempt,
	)
	if err != nil {
		return 0, false, fmt.Errorf("retry the page visit to %s: %w", url, err)
	}
	if standing.Holder == "" {
		return 0, false, fmt.Errorf("retry the page visit to %s: %w", url, errUnclaimedPageVisit)
	}
	return standing.Attempts, retried, nil
}

func (c *Claims) spendAttempt(claim pageVisitClaim) (pageVisitClaim, bool) {
	if claim.Holder == "" || claim.Attempts >= c.limits.MaxAttemptsPerURL {
		return claim, false
	}
	claim.Attempts++
	return claim, true
}

type pageVisitClaim struct {
	Holder    string `json:"Holder"`
	Deferrals int    `json:"Deferrals"`
	Attempts  int    `json:"Attempts"`
}

func pageVisitKeyOf(orderID string, url canonicalurl.CanonicalURL) string {
	return jetstreamrecord.KeyOf(orderID, url.String())
}
