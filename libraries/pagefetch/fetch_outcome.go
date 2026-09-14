package pagefetch

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type FetchOutcome struct {
	Status         FetchStatus
	DeferFor       time.Duration
	Page           FetchedPage
	RedirectTarget canonicalurl.CanonicalURL
	Version        PageVersion
	FailureCause   error
}
