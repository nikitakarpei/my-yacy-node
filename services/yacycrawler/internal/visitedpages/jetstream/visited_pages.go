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
) (pagevisit.LastPageVisit, bool) {
	lastVisit, visited, err := pages.readLastPageVisit(ctx, canonicalURL)
	if err != nil {
		pages.observer.LastPageVisitNotRead(ctx, canonicalURL, err)
		return pagevisit.LastPageVisit{}, false
	}
	return lastVisit, visited
}

func (pages *VisitedPages) readLastPageVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (pagevisit.LastPageVisit, bool, error) {
	entry, err := pages.bucket.Get(ctx, lastPageVisitKeyOf(canonicalURL))
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return pagevisit.LastPageVisit{}, false, nil
	}
	if err != nil {
		return pagevisit.LastPageVisit{}, false, fmt.Errorf(
			"get last page visit %s: %w", canonicalURL, err,
		)
	}
	record, err := unmarshalLastPageVisit(entry.Value())
	if err != nil {
		return pagevisit.LastPageVisit{}, false, err
	}
	return pagevisit.LastPageVisit{
		VisitedAt: record.VisitedAt,
		Version: pagefetch.PageVersion{
			EntityTag:  record.EntityTag,
			ModifiedAt: record.ModifiedAt,
		},
	}, true, nil
}

func lastPageVisitKeyOf(canonicalURL canonicalurl.CanonicalURL) string {
	return jetstreamrecord.KeyOf(canonicalURL.String())
}

func (pages *VisitedPages) RecordPageVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	if err := pages.putLastPageVisit(ctx, canonicalURL, version); err != nil {
		pages.observer.PageVisitNotRecorded(ctx, canonicalURL, err)
	}
}

func (pages *VisitedPages) putLastPageVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) error {
	payload, err := marshalLastPageVisit(lastPageVisit{
		VisitedAt:  pages.clock.Now(),
		EntityTag:  version.EntityTag,
		ModifiedAt: version.ModifiedAt,
	})
	if err != nil {
		return err
	}
	if _, err := pages.bucket.Put(ctx, lastPageVisitKeyOf(canonicalURL), payload); err != nil {
		return fmt.Errorf("put last page visit %s: %w", canonicalURL, err)
	}
	return nil
}
