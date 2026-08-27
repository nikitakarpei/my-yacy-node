// Package jetstream claims each URL of a crawl order for exactly one worker,
// and holds what that URL has spent of its deferrals, its attempts, and its
// host's page allowance.
package jetstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	BucketName        = "YACY_VISIT_CLAIMS"
	contendedAttempts = 5
)

var errContended = errors.New("another worker wrote the same key first")

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
	bucket jetstream.KeyValue
	config Config
}

func New(bucket jetstream.KeyValue, config Config) *Claims {
	return &Claims{bucket: bucket, config: config}
}

func (c *Claims) Claim(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	data, err := marshalVisitClaim(visitClaim{})
	if err != nil {
		return false, err
	}
	_, err = c.bucket.Create(ctx, visitKeyOf(orderID, url), data)
	if errors.Is(err, jetstream.ErrKeyExists) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim visit %s: %w", url, err)
	}
	return true, nil
}

func (c *Claims) Defer(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	return c.spendVisitClaim(ctx, orderID, url, func(claim *visitClaim) bool {
		if claim.Deferrals >= c.config.MaxDeferralsPerURL {
			return false
		}
		claim.Deferrals++
		return true
	})
}

func (c *Claims) Retry(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
) (int, bool, error) {
	attempt := 0
	retried, err := c.spendVisitClaim(ctx, orderID, url, func(claim *visitClaim) bool {
		if claim.Attempts >= c.config.MaxAttemptsPerURL {
			return false
		}
		claim.Attempts++
		attempt = claim.Attempts
		return true
	})
	return attempt, retried, err
}

func (c *Claims) SpendHostPage(
	ctx context.Context,
	orderID string,
	host string,
	maxPages int,
) (bool, error) {
	if maxPages == yacycrawlcontract.UnlimitedPagesPerHost {
		return true, nil
	}
	key := hostKeyOf(orderID, host)
	for range contendedAttempts {
		spent, err := c.spendOneHostPage(ctx, key, maxPages)
		if errors.Is(err, errContended) {
			continue
		}
		if err != nil {
			return false, err
		}
		return spent, nil
	}
	return false, fmt.Errorf("spend a page of host %s: %w", host, errContended)
}

func (c *Claims) spendOneHostPage(
	ctx context.Context,
	key string,
	maxPages int,
) (bool, error) {
	entry, err := c.bucket.Get(ctx, key)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return c.openHostPages(ctx, key)
	}
	if err != nil {
		return false, fmt.Errorf("read host pages: %w", err)
	}
	pages, err := unmarshalHostPages(entry.Value())
	if err != nil {
		return false, err
	}
	if pages.Pages >= maxPages {
		return false, nil
	}
	pages.Pages++
	data, err := marshalHostPages(pages)
	if err != nil {
		return false, err
	}
	if _, err := c.bucket.Update(ctx, key, data, entry.Revision()); err != nil {
		return false, errContended
	}
	return true, nil
}

func (c *Claims) openHostPages(ctx context.Context, key string) (bool, error) {
	data, err := marshalHostPages(hostPages{Pages: 1})
	if err != nil {
		return false, err
	}
	if _, err := c.bucket.Create(ctx, key, data); err != nil {
		return false, errContended
	}
	return true, nil
}

func (c *Claims) spendVisitClaim(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
	take func(*visitClaim) bool,
) (bool, error) {
	key := visitKeyOf(orderID, url)
	for range contendedAttempts {
		taken, err := c.takeOnce(ctx, key, take)
		if errors.Is(err, errContended) {
			continue
		}
		if err != nil {
			return false, err
		}
		return taken, nil
	}
	return false, fmt.Errorf("spend the visit claim for %s: %w", url, errContended)
}

func (c *Claims) takeOnce(
	ctx context.Context,
	key string,
	take func(*visitClaim) bool,
) (bool, error) {
	entry, err := c.bucket.Get(ctx, key)
	if err != nil {
		return false, fmt.Errorf("read visit claim: %w", err)
	}
	claim, err := unmarshalVisitClaim(entry.Value())
	if err != nil {
		return false, err
	}
	if !take(&claim) {
		return false, nil
	}
	data, err := marshalVisitClaim(claim)
	if err != nil {
		return false, err
	}
	if _, err := c.bucket.Update(ctx, key, data, entry.Revision()); err != nil {
		return false, errContended
	}
	return true, nil
}

type visitClaim struct {
	Deferrals int `json:"Deferrals"`
	Attempts  int `json:"Attempts"`
}

func marshalVisitClaim(claim visitClaim) ([]byte, error) {
	data, err := json.Marshal(claim)
	if err != nil {
		return nil, fmt.Errorf("marshal visit claim: %w", err)
	}
	return data, nil
}

func unmarshalVisitClaim(data []byte) (visitClaim, error) {
	var claim visitClaim
	if err := json.Unmarshal(data, &claim); err != nil {
		return visitClaim{}, fmt.Errorf("unmarshal visit claim: %w", err)
	}
	return claim, nil
}

type hostPages struct {
	Pages int `json:"Pages"`
}

func marshalHostPages(pages hostPages) ([]byte, error) {
	data, err := json.Marshal(pages)
	if err != nil {
		return nil, fmt.Errorf("marshal host pages: %w", err)
	}
	return data, nil
}

func unmarshalHostPages(data []byte) (hostPages, error) {
	var pages hostPages
	if err := json.Unmarshal(data, &pages); err != nil {
		return hostPages{}, fmt.Errorf("unmarshal host pages: %w", err)
	}
	return pages, nil
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
