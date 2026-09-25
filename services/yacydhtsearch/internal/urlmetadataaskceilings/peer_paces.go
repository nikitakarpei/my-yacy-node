package urlmetadataaskceilings

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerPaces interface {
	Read(ctx context.Context, address string) yacymodel.Optional[time.Duration]
	Update(
		ctx context.Context,
		address string,
		updated func(pace yacymodel.Optional[time.Duration]) time.Duration,
	) time.Duration
}
