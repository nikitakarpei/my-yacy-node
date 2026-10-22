package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type PageVisitRecords interface {
	LastPageVisitOf(
		tx *vault.Txn,
		canonicalURL canonicalurl.CanonicalURL,
	) (LastPageVisit, bool, error)
	RecordPageVisit(
		tx *vault.Txn,
		canonicalURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	) error
}

type LastPageVisit struct {
	VisitedAt time.Time
	Version   pagefetch.PageVersion
}
