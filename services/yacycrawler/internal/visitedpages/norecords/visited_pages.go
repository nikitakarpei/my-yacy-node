// Package norecords keeps no page visit, so no page was ever visited.
package norecords

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type VisitedPages struct{}

func (VisitedPages) LastPageVisitOf(
	context.Context,
	canonicalurl.CanonicalURL,
) (pagevisit.PageVisit, bool, error) {
	return pagevisit.PageVisit{}, false, nil
}

func (VisitedPages) RecordPageVisit(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.PageVersion,
) {
}
