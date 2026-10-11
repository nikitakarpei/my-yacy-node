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

type PageVisitRecords struct {
	bucket   jetstream.KeyValue
	clock    Clock
	observer PageVisitRecordObserver
}

func New(
	bucket jetstream.KeyValue,
	clock Clock,
	observer PageVisitRecordObserver,
) *PageVisitRecords {
	return &PageVisitRecords{bucket: bucket, clock: clock, observer: observer}
}

func (records *PageVisitRecords) LastPageVisitOf(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (pagevisit.LastPageVisit, bool) {
	lastVisit, visited, err := records.readLastPageVisit(ctx, canonicalURL)
	if err != nil {
		records.observer.LastPageVisitNotRead(ctx, canonicalURL, err)
		return pagevisit.LastPageVisit{}, false
	}
	return lastVisit, visited
}

func (records *PageVisitRecords) readLastPageVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (pagevisit.LastPageVisit, bool, error) {
	entry, err := records.bucket.Get(ctx, lastPageVisitKeyOf(canonicalURL))
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

func (records *PageVisitRecords) RecordPageVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	if err := records.putLastPageVisit(ctx, canonicalURL, version); err != nil {
		records.observer.PageVisitNotRecorded(ctx, canonicalURL, err)
	}
}

func (records *PageVisitRecords) putLastPageVisit(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) error {
	payload, err := marshalLastPageVisit(lastPageVisit{
		VisitedAt:  records.clock.Now(),
		EntityTag:  version.EntityTag,
		ModifiedAt: version.ModifiedAt,
	})
	if err != nil {
		return err
	}
	if _, err := records.bucket.Put(ctx, lastPageVisitKeyOf(canonicalURL), payload); err != nil {
		return fmt.Errorf("put last page visit %s: %w", canonicalURL, err)
	}
	return nil
}
