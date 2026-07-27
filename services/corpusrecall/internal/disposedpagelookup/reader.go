package disposedpagelookup

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type DisposedPages interface {
	Get(ctx context.Context, key string) (jetstream.KeyValueEntry, error)
}

type Reader struct {
	disposed DisposedPages
}

func NewReader(disposed DisposedPages) *Reader {
	return &Reader{disposed: disposed}
}

// Revision returns the disposal record's revision for a URL, or zero if the crawler
// has not disposed of it.
func (r *Reader) Revision(ctx context.Context, canonicalURL string) (uint64, error) {
	entry, err := r.disposed.Get(ctx, yacycrawlcontract.DisposedPageKey(canonicalURL))
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("look up disposal for %q: %w", canonicalURL, err)
	}
	return entry.Revision(), nil
}
