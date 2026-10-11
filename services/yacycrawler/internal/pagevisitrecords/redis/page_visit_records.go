// Package redis keeps the last page visit of every page for the retention of page
// visits, with the page version that visit saw.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redisrecord"
)

const recordKind = "YACY_PAGE_VISITS"

type Clock interface {
	Now() time.Time
}

type PageVisitRecords struct {
	client    redis.UniversalClient
	retention time.Duration
	clock     Clock
	observer  PageVisitRecordObserver
}

func New(
	client redis.UniversalClient,
	retention time.Duration,
	clock Clock,
	observer PageVisitRecordObserver,
) *PageVisitRecords {
	return &PageVisitRecords{client: client, retention: retention, clock: clock, observer: observer}
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
	payload, err := records.client.Get(ctx, lastPageVisitKeyOf(canonicalURL)).Bytes()
	if errors.Is(err, redis.Nil) {
		return pagevisit.LastPageVisit{}, false, nil
	}
	if err != nil {
		return pagevisit.LastPageVisit{}, false, fmt.Errorf(
			"get last page visit %s: %w", canonicalURL, err,
		)
	}
	record, err := unmarshalLastPageVisit(payload)
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
	return redisrecord.KeyOf(recordKind, canonicalURL.String())
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
	err = records.client.Set(ctx, lastPageVisitKeyOf(canonicalURL), payload, records.retention).
		Err()
	if err != nil {
		return fmt.Errorf("put last page visit %s: %w", canonicalURL, err)
	}
	return nil
}
