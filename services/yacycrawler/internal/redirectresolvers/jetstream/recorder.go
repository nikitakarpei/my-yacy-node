package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type Recorder struct {
	bucket jetstream.KeyValue
}

func New(bucket jetstream.KeyValue) *Recorder {
	return &Recorder{bucket: bucket}
}

func (r *Recorder) Record(
	ctx context.Context,
	requested, canonical yacycrawlcontract.CanonicalURL,
) error {
	if requested == canonical {
		return nil
	}
	key := yacycrawlcontract.RedirectResolutionKey(requested)
	if _, err := r.bucket.Put(ctx, key, []byte(canonical.String())); err != nil {
		return fmt.Errorf("put redirect resolution %s: %w", requested, err)
	}
	return nil
}
