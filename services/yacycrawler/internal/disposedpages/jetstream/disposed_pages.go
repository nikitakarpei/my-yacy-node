// Package jetstream holds the pages the crawler disposed of, and why, in a key-value bucket.
package jetstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

const orderedMarkFormat = "%020d"

type DisposedPages struct {
	bucket jetstream.KeyValue
}

func NewDisposedPages(bucket jetstream.KeyValue) *DisposedPages {
	return &DisposedPages{bucket: bucket}
}

func (p *DisposedPages) Record(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	reason disposal.Reason,
) error {
	key := yacycrawlcontract.DisposedPageKey(canonicalURL)
	if _, err := p.bucket.Put(ctx, key, []byte(reason)); err != nil {
		return fmt.Errorf("put disposed page %s: %w", canonicalURL, err)
	}
	return nil
}

func (p *DisposedPages) DisposedPageOf(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (disposal.DisposedPage, bool, error) {
	key := yacycrawlcontract.DisposedPageKey(canonicalURL)
	entry, err := p.bucket.Get(ctx, key)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return disposal.DisposedPage{}, false, nil
	}
	if err != nil {
		return disposal.DisposedPage{}, false, fmt.Errorf(
			"get disposed page %s: %w", canonicalURL, err,
		)
	}
	return disposal.DisposedPage{
		Mark:   disposal.Mark(fmt.Sprintf(orderedMarkFormat, entry.Revision())),
		Reason: disposal.Reason(entry.Value()),
	}, true, nil
}
