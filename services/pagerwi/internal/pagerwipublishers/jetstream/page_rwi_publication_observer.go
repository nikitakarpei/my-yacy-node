package jetstream

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageRWIPublicationObserver interface {
	PageRWIPublished(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	PageRWIEncodingFailed(ctx context.Context, pageURL canonicalurl.CanonicalURL, cause error)
	PageRWIPublishingFailed(ctx context.Context, pageURL canonicalurl.CanonicalURL, cause error)
}
