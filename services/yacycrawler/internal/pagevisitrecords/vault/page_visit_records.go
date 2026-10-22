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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const pageVisitsBucket = "page_visit_records"

var pageKeys = vault.SingleKey(vaultkeyparts.URLHash).KeyLayout()

type Clock interface {
	Now() time.Time
}

type PageVisitRecords struct {
	storage    *vault.Vault
	lastVisits *vault.ExpiringCollection[urlhash.URLHash, lastPageVisit]
	clock      Clock
	observer   PageVisitRecordObserver
}

func New(
	ctx context.Context,
	storage *vault.Vault,
	clock Clock,
	keptFor time.Duration,
	observer PageVisitRecordObserver,
) (*PageVisitRecords, error) {
	lastVisits, err := storage.RegisterExpiringCollection(
		ctx,
		pageVisitsBucket,
		pageKeys,
		lastPageVisitCodec{},
		keptFor,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", pageVisitsBucket, err)
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
	var record lastPageVisit
	var visited bool
	if err := records.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		record, visited, err = records.lastVisits.Get(tx, urlhash.Of(canonicalURL))
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
	visitedAt := records.clock.Now()
	if err := records.storage.Update(ctx, func(tx *vault.Txn) error {
		return records.lastVisits.Put(tx, urlhash.Of(canonicalURL), lastPageVisit{
			visitedAt:  visitedAt,
			entityTag:  version.EntityTag,
			modifiedAt: version.ModifiedAt,
		})
	}); err != nil {
		return fmt.Errorf("put last page visit %s: %w", canonicalURL, err)
	}
	return nil
}
