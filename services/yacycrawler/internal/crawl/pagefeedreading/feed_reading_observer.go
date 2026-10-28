package pagefeedreading

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type FeedReadingObserver interface {
	FeedUnreadable(ctx context.Context, pageURL canonicalurl.CanonicalURL, cause error)
}
