package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type RecrawlRule interface {
	DecisionFor(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
	) (RecrawlDecision, error)
	RecordVisit(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	) error
}

type RecrawlDecision struct {
	Due     bool
	Version pagefetch.PageVersion
}

type BestEffortRecrawlRule struct {
	inner    RecrawlRule
	observer PageVisitRecordObserver
}

func NewBestEffortRecrawlRule(
	inner RecrawlRule,
	observer PageVisitRecordObserver,
) *BestEffortRecrawlRule {
	return &BestEffortRecrawlRule{inner: inner, observer: observer}
}

func (r *BestEffortRecrawlRule) DecisionFor(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) (RecrawlDecision, error) {
	return r.inner.DecisionFor(ctx, pageURL)
}

func (r *BestEffortRecrawlRule) RecordVisit(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) error {
	if err := r.inner.RecordVisit(ctx, pageURL, version); err != nil {
		r.observer.PageVisitNotRecorded(ctx, pageURL, err)
	}
	return nil
}
