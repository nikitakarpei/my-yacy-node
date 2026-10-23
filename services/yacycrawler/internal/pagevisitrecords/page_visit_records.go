// Package pagevisitrecords keeps the last page visit of every page for the retention of
// page visits, with the page version that visit saw, or keeps none.
package pagevisitrecords

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

type Clock interface {
	Now() time.Time
}

type PageVisitRecords struct {
	lastVisits *vault.ExpiringCollection[urlhash.URLHash, lastPageVisit]
	clock      Clock
}

func Open(
	ctx context.Context,
	storage *vault.Vault,
	clock Clock,
	keptFor time.Duration,
) (*PageVisitRecords, error) {
	lastVisits, err := registerPageVisitRecords(ctx, storage, keptFor)
	if err != nil {
		return nil, err
	}
	return &PageVisitRecords{lastVisits: lastVisits, clock: clock}, nil
}

type lastPageVisit struct {
	visitedAt  time.Time
	entityTag  string
	modifiedAt time.Time
}

func (records *PageVisitRecords) LastPageVisitOf(
	tx *vault.Txn,
	canonicalURL canonicalurl.CanonicalURL,
) (pagevisit.LastPageVisit, bool, error) {
	record, visited, err := records.lastVisits.Get(tx, urlhash.Of(canonicalURL))
	if err != nil {
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
	tx *vault.Txn,
	canonicalURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) error {
	if err := records.lastVisits.Put(tx, urlhash.Of(canonicalURL), lastPageVisit{
		visitedAt:  records.clock.Now(),
		entityTag:  version.EntityTag,
		modifiedAt: version.ModifiedAt,
	}); err != nil {
		return fmt.Errorf("put last page visit %s: %w", canonicalURL, err)
	}
	return nil
}
