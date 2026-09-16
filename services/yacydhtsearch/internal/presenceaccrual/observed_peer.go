package presenceaccrual

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
)

type ObservedPeer struct {
	peeranswerhistory.PeerAtAddress
	FirstAnsweredAt  time.Time
	LatestAnsweredAt time.Time
	Presence         time.Duration
}
