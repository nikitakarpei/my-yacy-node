package jetstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitclaim"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
)

const BucketName = "YACY_VISIT_CLAIMS"

var errUnclaimedVisit = errors.New("no holder claimed the visit")

type BucketSpec struct {
	MaxBytes  int64
	Retention time.Duration
}

func Ensure(ctx context.Context, js jetstream.JetStream, spec BucketSpec) error {
	if _, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   BucketName,
		MaxBytes: spec.MaxBytes,
		TTL:      spec.Retention,
	}); err != nil {
		return fmt.Errorf("ensure visit claim bucket: %w", err)
	}
	return nil
}

type Config struct {
	MaxDeferralsPerURL int
	MaxAttemptsPerURL  int
}

type Claims struct {
	visitClaims *jetstreamrecord.Records[visitClaim]
	hostPages   *jetstreamrecord.Records[hostPages]
	config      Config
}

func New(bucket jetstream.KeyValue, config Config) *Claims {
	return &Claims{
		visitClaims: jetstreamrecord.New[visitClaim](bucket),
		hostPages:   jetstreamrecord.New[hostPages](bucket),
		config:      config,
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
	if claim.Holder == "" || claim.Deferrals >= c.config.MaxDeferralsPerURL {
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
	if claim.Holder == "" || claim.Attempts >= c.config.MaxAttemptsPerURL {
		return claim, false
	}
	claim.Attempts++
	return claim, true
}

func (c *Claims) SpendHostPage(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
	host string,
	maxPages int,
) (bool, error) {
	if maxPages == yacycrawlcontract.UnlimitedPagesPerHost {
		return true, nil
	}
	alreadySpent, err := c.hostPageSpentOnVisit(ctx, orderID, url)
	if err != nil {
		return false, err
	}
	if alreadySpent {
		return true, nil
	}
	spent, err := c.takeHostPage(ctx, orderID, host, maxPages)
	if err != nil {
		return false, err
	}
	if !spent {
		return false, nil
	}
	if err := c.markHostPageSpentOnVisit(ctx, orderID, url); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Claims) hostPageSpentOnVisit(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	claim, err := c.visitClaims.RecordAt(ctx, visitKeyOf(orderID, url))
	if err != nil {
		return false, fmt.Errorf("read the host page spent on the visit to %s: %w", url, err)
	}
	return claim.HostPageSpent, nil
}

func (c *Claims) takeHostPage(
	ctx context.Context,
	orderID string,
	host string,
	maxPages int,
) (bool, error) {
	_, taken, err := c.hostPages.Revise(
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

func (c *Claims) markHostPageSpentOnVisit(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) error {
	_, _, err := c.visitClaims.Revise(ctx, visitKeyOf(orderID, url), markHostPageSpent)
	if err != nil {
		return fmt.Errorf("mark the host page spent on the visit to %s: %w", url, err)
	}
	return nil
}

func markHostPageSpent(claim visitClaim) (visitClaim, bool) {
	if claim.HostPageSpent {
		return claim, false
	}
	claim.HostPageSpent = true
	return claim, true
}

type visitClaim struct {
	Holder        string `json:"Holder"`
	Deferrals     int    `json:"Deferrals"`
	Attempts      int    `json:"Attempts"`
	HostPageSpent bool   `json:"HostPageSpent"`
}

type hostPages struct {
	Pages int `json:"Pages"`
}

func visitKeyOf(orderID string, url canonicalurl.CanonicalURL) string {
	return digestOf(orderID) + ".url." + digestOf(url.String())
}

func hostKeyOf(orderID string, host string) string {
	return digestOf(orderID) + ".host." + digestOf(host)
}

func digestOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
