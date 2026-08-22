package frontier

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PendingVisit struct {
	URL       canonicalurl.CanonicalURL
	Depth     int
	deferrals int
	attempts  int
	notBefore time.Time
}
