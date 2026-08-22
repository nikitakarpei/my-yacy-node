// Package jetstream holds where the crawler resolved each requested URL to, in a
// key-value bucket.
package jetstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type RedirectResolutions struct {
	bucket jetstream.KeyValue
}

func NewRedirectResolutions(bucket jetstream.KeyValue) *RedirectResolutions {
	return &RedirectResolutions{bucket: bucket}
}

func (r *RedirectResolutions) Record(
	ctx context.Context,
	requested, canonical canonicalurl.CanonicalURL,
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

func (r *RedirectResolutions) ResolvedURLOf(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (canonicalurl.CanonicalURL, error) {
	key := yacycrawlcontract.RedirectResolutionKey(canonicalURL)
	entry, err := r.bucket.Get(ctx, key)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return canonicalURL, nil
	}
	if err != nil {
		return canonicalurl.CanonicalURL{}, fmt.Errorf(
			"get redirect resolution %s: %w", canonicalURL, err,
		)
	}
	resolvedURL, err := canonicalurl.CanonicalURLOf(string(entry.Value()))
	if err != nil {
		return canonicalurl.CanonicalURL{}, fmt.Errorf(
			"get redirect resolution %s: %w", canonicalURL, err,
		)
	}
	return resolvedURL, nil
}
