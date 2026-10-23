package pagevisitrecords

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type None struct{}

func (None) LastPageVisitOf(
	*vault.Txn,
	canonicalurl.CanonicalURL,
) (pagevisit.LastPageVisit, bool, error) {
	return pagevisit.LastPageVisit{}, false, nil
}

func (None) RecordPageVisit(*vault.Txn, canonicalurl.CanonicalURL, pagefetch.PageVersion) error {
	return nil
}
