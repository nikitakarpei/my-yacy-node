package frontier

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type PendingVisit struct {
	URL       yacycrawlcontract.CanonicalURL
	Depth     int
	deferrals int
	attempts  int
	notBefore time.Time
}
