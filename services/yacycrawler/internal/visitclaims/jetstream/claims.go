package jetstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitclaim"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
)

const BucketName = "YACY_VISIT_CLAIMS"

var errUnclaimedVisit = errors.New("no holder claimed the visit")

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
	visitClaims *jetstreamrecord.Records[visitClaim]
	limits      ClaimLimits
}

func New(bucket jetstream.KeyValue, limits ClaimLimits) *Claims {
	return &Claims{
		visitClaims: jetstreamrecord.New[visitClaim](bucket),
		limits:      limits,
	}
}

func (c *Claims) Claim(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
	holder string,
) (visitclaim.Claim, error) {
	standing, taken, err := c.visitClaims.Revise(
		ctx, visitKeyOf(orderID, url), takenBy(holder),
	)
	if err != nil {
		return visitclaim.Unanswered, fmt.Errorf("claim the visit to %s: %w", url, err)
	}
	if taken {
		return visitclaim.Taken, nil
	}
	if standing.Holder == holder {
		return visitclaim.Resumed, nil
	}
	return visitclaim.HeldElsewhere, nil
}

func takenBy(holder string) func(visitClaim) (visitClaim, bool) {
	return func(claim visitClaim) (visitClaim, bool) {
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
	standing, deferred, err := c.visitClaims.Revise(
		ctx, visitKeyOf(orderID, url), c.spendDeferral,
	)
	if err != nil {
		return false, fmt.Errorf("defer the visit to %s: %w", url, err)
	}
	if standing.Holder == "" {
		return false, fmt.Errorf("defer the visit to %s: %w", url, errUnclaimedVisit)
	}
	return deferred, nil
}

func (c *Claims) spendDeferral(claim visitClaim) (visitClaim, bool) {
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
	standing, retried, err := c.visitClaims.Revise(
		ctx, visitKeyOf(orderID, url), c.spendAttempt,
	)
	if err != nil {
		return 0, false, fmt.Errorf("retry the visit to %s: %w", url, err)
	}
	if standing.Holder == "" {
		return 0, false, fmt.Errorf("retry the visit to %s: %w", url, errUnclaimedVisit)
	}
	return standing.Attempts, retried, nil
}

func (c *Claims) spendAttempt(claim visitClaim) (visitClaim, bool) {
	if claim.Holder == "" || claim.Attempts >= c.limits.MaxAttemptsPerURL {
		return claim, false
	}
	claim.Attempts++
	return claim, true
}

type visitClaim struct {
	Holder    string `json:"Holder"`
	Deferrals int    `json:"Deferrals"`
	Attempts  int    `json:"Attempts"`
}

func visitKeyOf(orderID string, url canonicalurl.CanonicalURL) string {
	return jetstreamrecord.KeyOf(orderID, url.String())
}
