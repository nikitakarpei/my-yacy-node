package pagevisit

import (
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type RecrawlRule interface {
	RecrawlOf(lastVisit LastPageVisit, recrawlGrace yacycrawlcontract.RecrawlGrace) Recrawl
}

type Recrawl struct {
	Due          bool
	KnownVersion pagefetch.PageVersion
}
