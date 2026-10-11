// Package norecords keeps no page visit, so no page was ever visited.
package norecords

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type PageVisitRecords struct{}

func (PageVisitRecords) LastPageVisitOf(
	context.Context,
	canonicalurl.CanonicalURL,
) (pagevisit.LastPageVisit, bool) {
	return pagevisit.LastPageVisit{}, false
}

func (PageVisitRecords) RecordPageVisit(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.PageVersion,
) {
}
