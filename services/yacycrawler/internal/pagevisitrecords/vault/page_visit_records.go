// Package vault keeps in a vault the last page visit of every page for the retention
// of page visits, with the page version that visit saw.
package vault

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlkeyparts"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/retainedrecords"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

const pageVisitsBucket = "page_visit_records"

var pageKeys = vault.SingleKey(crawlkeyparts.URLHash).KeyLayout()

type Clock interface {
	Now() time.Time
}

type PageVisitRecords struct {
	storage    *vault.Vault
	lastVisits *retainedrecords.Records[urlhash.URLHash, lastPageVisit]
	clock      Clock
	observer   PageVisitRecordObserver
}

//nolint:revive // the records name the vault, its retention and the observer they report to
func New(
	ctx context.Context,
	storage *vault.Vault,
	retention *retainedrecords.Retention,
	clock Clock,
	keptFor time.Duration,
	observer PageVisitRecordObserver,
) (*PageVisitRecords, error) {
	lastVisits, err := retention.Register(
		ctx,
		pageVisitsBucket,
		pageKeys,
		lastPageVisitCodec{},
		keptFor,
	)
	if err != nil {
		return nil, err
	}
	return &PageVisitRecords{
		storage:    storage,
		lastVisits: lastVisits,
		clock:      clock,
		observer:   observer,
	}, nil
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
	now := records.clock.Now()
	var record lastPageVisit
	var visited bool
	if err := records.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		record, visited, err = records.lastVisits.Get(tx, urlhash.Of(canonicalURL), now)
		return err
	}); err != nil {
		return pagevisit.LastPageVisit{}, false, fmt.Errorf(
			"get last page visit %s: %w", canonicalURL, err,
		)
	}
	return pagevisit.LastPageVisit{
		VisitedAt: record.visitedAt,
		Version: pagefetch.PageVersion{
			EntityTag:  record.entityTag,
			ModifiedAt: record.modifiedAt,
		},
	}, visited, nil
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
	now := records.clock.Now()
	if err := records.storage.Update(ctx, func(tx *vault.Txn) error {
		return records.lastVisits.Put(tx, urlhash.Of(canonicalURL), lastPageVisit{
			visitedAt:  now,
			entityTag:  version.EntityTag,
			modifiedAt: version.ModifiedAt,
		}, now)
	}); err != nil {
		return fmt.Errorf("put last page visit %s: %w", canonicalURL, err)
	}
	return nil
}
