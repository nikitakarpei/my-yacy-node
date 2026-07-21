package redirectlookup

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type RedirectResolution interface {
	Get(ctx context.Context, key string) (jetstream.KeyValueEntry, error)
}

type Reader struct {
	redirects RedirectResolution
}

func NewReader(redirects RedirectResolution) *Reader {
	return &Reader{redirects: redirects}
}

func (r *Reader) Resolve(ctx context.Context, canonicalURL string) (string, error) {
	entry, err := r.redirects.Get(ctx, yacycrawlcontract.RedirectResolutionKey(canonicalURL))
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return canonicalURL, nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve redirect for %q: %w", canonicalURL, err)
	}
	return string(entry.Value()), nil
}
