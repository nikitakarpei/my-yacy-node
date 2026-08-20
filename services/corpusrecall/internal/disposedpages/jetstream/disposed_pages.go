// Package jetstream reads the pages the crawler disposed of from a key-value bucket.
package jetstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const orderedDisposalMarkFormat = "%020d"

type DisposedPages struct {
	bucket jetstream.KeyValue
}

func NewDisposedPages(bucket jetstream.KeyValue) *DisposedPages {
	return &DisposedPages{bucket: bucket}
}

func (p *DisposedPages) DisposalMarkOf(
	ctx context.Context,
	canonicalURL yacycrawlcontract.CanonicalURL,
) (recall.DisposalMark, error) {
	return disposalMarkIn(ctx, p.bucket, canonicalURL)
}

func disposalMarkIn(
	ctx context.Context,
	bucket jetstream.KeyValue,
	canonicalURL yacycrawlcontract.CanonicalURL,
) (recall.DisposalMark, error) {
	key := yacycrawlcontract.DisposedPageKey(canonicalURL)
	entry, err := bucket.Get(ctx, key)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("look up disposal under %q: %w", key, err)
	}
	return recall.DisposalMark(
		fmt.Sprintf(orderedDisposalMarkFormat, entry.Revision()),
	), nil
}

func (p *DisposedPages) DisposalOccurredSince(
	ctx context.Context,
	canonicalURL yacycrawlcontract.CanonicalURL,
	mark recall.DisposalMark,
) (bool, error) {
	current, err := disposalMarkIn(ctx, p.bucket, canonicalURL)
	if err != nil {
		return false, err
	}
	return current > mark, nil
}
