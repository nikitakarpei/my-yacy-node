package pagevisit

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type VisitedPages interface {
	LastPageVisitOf(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
	) (LastPageVisit, bool)
	RecordPageVisit(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	)
}

type LastPageVisit struct {
	VisitedAt time.Time
	Version   pagefetch.PageVersion
}
