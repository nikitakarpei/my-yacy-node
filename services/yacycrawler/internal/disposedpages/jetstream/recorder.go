// Package jetstream records pages the crawler disposed of without publishing.
package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type Recorder struct {
	bucket jetstream.KeyValue
}

func New(bucket jetstream.KeyValue) *Recorder {
	return &Recorder{bucket: bucket}
}

func (r *Recorder) Record(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) error {
	key := yacycrawlcontract.DisposedPageKey(canonicalURL)
	if _, err := r.bucket.Put(ctx, key, nil); err != nil {
		return fmt.Errorf("put disposed page %s: %w", canonicalURL, err)
	}
	return nil
}
