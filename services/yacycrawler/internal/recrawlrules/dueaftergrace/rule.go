// Package dueaftergrace decides a page is due once a configured grace
// window has elapsed since its last visit, and supplies the page version
// that visit recorded.
package dueaftergrace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
)

type Clock interface {
	Now() time.Time
}

type Rule struct {
	bucket jetstream.KeyValue
	clock  Clock
	grace  time.Duration
}

func New(bucket jetstream.KeyValue, clock Clock, grace time.Duration) *Rule {
	return &Rule{bucket: bucket, clock: clock, grace: grace}
}

func (r *Rule) DecisionFor(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (pagevisit.RecrawlDecision, error) {
	entry, err := r.bucket.Get(ctx, pageVisitKeyOf(canonicalURL))
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return pagevisit.RecrawlDecision{Due: true}, nil
	}
	if err != nil {
		return pagevisit.RecrawlDecision{}, fmt.Errorf("get page visit %s: %w", canonicalURL, err)
	}
	record, err := unmarshalPageVisit(entry.Value())
	if err != nil {
		return pagevisit.RecrawlDecision{}, err
	}
	return pagevisit.RecrawlDecision{
		Due: r.clock.Now().Sub(record.VisitedAt) >= r.grace,
		Version: pagefetch.PageVersion{
			EntityTag:  record.EntityTag,
			ModifiedAt: record.ModifiedAt,
		},
	}, nil
}

func pageVisitKeyOf(canonicalURL canonicalurl.CanonicalURL) string {
	return jetstreamrecord.KeyOf(canonicalURL.String())
}

func (r *Rule) RecordVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) error {
	payload, err := marshalPageVisit(pageVisit{
		VisitedAt:  r.clock.Now(),
		EntityTag:  version.EntityTag,
		ModifiedAt: version.ModifiedAt,
	})
	if err != nil {
		return err
	}
	if _, err := r.bucket.Put(ctx, pageVisitKeyOf(canonicalURL), payload); err != nil {
		return fmt.Errorf("put page visit %s: %w", canonicalURL, err)
	}
	return nil
}
