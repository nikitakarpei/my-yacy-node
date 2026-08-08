// Package jetstream reads the redirect resolutions the crawler recorded in a key-value bucket.
package jetstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type RedirectResolutions struct {
	bucket jetstream.KeyValue
}

func NewRedirectResolutions(bucket jetstream.KeyValue) *RedirectResolutions {
	return &RedirectResolutions{bucket: bucket}
}

func (r *RedirectResolutions) ResolvedURLOf(
	ctx context.Context,
	canonicalURL string,
) (string, error) {
	entry, err := r.bucket.Get(ctx, yacycrawlcontract.RedirectResolutionKey(canonicalURL))
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return canonicalURL, nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve redirect for %q: %w", canonicalURL, err)
	}
	return string(entry.Value()), nil
}
