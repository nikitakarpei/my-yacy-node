package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type RecrawlRule interface {
	RecrawlOf(lastVisit LastPageVisit, recrawlGrace time.Duration) Recrawl
}

type Recrawl struct {
	Due          bool
	KnownVersion pagefetch.PageVersion
}
