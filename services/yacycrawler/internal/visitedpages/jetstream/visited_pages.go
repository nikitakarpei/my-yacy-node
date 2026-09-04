// Package jetstream keeps the last page visit of every page in a key-value
// bucket, with the page version that visit saw.
package jetstream

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

type VisitedPages struct {
	bucket   jetstream.KeyValue
	clock    Clock
	observer PageVisitRecordObserver
}

func New(
	bucket jetstream.KeyValue,
	clock Clock,
	observer PageVisitRecordObserver,
) *VisitedPages {
	return &VisitedPages{bucket: bucket, clock: clock, observer: observer}
}

func (pages *VisitedPages) LastPageVisitOf(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (pagevisit.PageVisit, bool) {
	lastVisit, visited, err := pages.readLastPageVisit(ctx, canonicalURL)
	if err != nil {
		pages.observer.LastPageVisitNotRead(ctx, canonicalURL, err)
		return pagevisit.PageVisit{}, false
	}
	return lastVisit, visited
}

func (pages *VisitedPages) readLastPageVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (pagevisit.PageVisit, bool, error) {
	entry, err := pages.bucket.Get(ctx, pageVisitKeyOf(canonicalURL))
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return pagevisit.PageVisit{}, false, nil
	}
	if err != nil {
		return pagevisit.PageVisit{}, false, fmt.Errorf(
			"get page visit %s: %w", canonicalURL, err,
		)
	}
	record, err := unmarshalPageVisit(entry.Value())
	if err != nil {
		return pagevisit.PageVisit{}, false, err
	}
	return pagevisit.PageVisit{
		VisitedAt: record.VisitedAt,
		Version: pagefetch.PageVersion{
			EntityTag:  record.EntityTag,
			ModifiedAt: record.ModifiedAt,
		},
	}, true, nil
}

func pageVisitKeyOf(canonicalURL canonicalurl.CanonicalURL) string {
	return jetstreamrecord.KeyOf(canonicalURL.String())
}

func (pages *VisitedPages) RecordPageVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	if err := pages.putPageVisit(ctx, canonicalURL, version); err != nil {
		pages.observer.PageVisitNotRecorded(ctx, canonicalURL, err)
	}
}

func (pages *VisitedPages) putPageVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) error {
	payload, err := marshalPageVisit(pageVisit{
		VisitedAt:  pages.clock.Now(),
		EntityTag:  version.EntityTag,
		ModifiedAt: version.ModifiedAt,
	})
	if err != nil {
		return err
	}
	if _, err := pages.bucket.Put(ctx, pageVisitKeyOf(canonicalURL), payload); err != nil {
		return fmt.Errorf("put page visit %s: %w", canonicalURL, err)
	}
	return nil
}
