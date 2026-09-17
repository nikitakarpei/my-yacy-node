// Package constant holds the hedge delay of a peer, one value from
// configuration that every peer shares.
package constant

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type HedgeDelay struct {
	delay time.Duration
}

func New(delay time.Duration) HedgeDelay {
	return HedgeDelay{delay: delay}
}

func (hedgeDelay HedgeDelay) HedgeDelayOf(
	_ context.Context,
	_ peerdirectory.AskablePeer,
) time.Duration {
	return hedgeDelay.delay
}
