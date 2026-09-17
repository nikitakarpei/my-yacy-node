package presenceaccrual

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
)

type ObservedPeer struct {
	probeanswerhistory.PeerAtAddress
	FirstAnsweredAt  time.Time
	LatestAnsweredAt time.Time
	Presence         time.Duration
}
