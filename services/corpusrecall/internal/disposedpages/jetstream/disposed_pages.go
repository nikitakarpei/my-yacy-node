// Package jetstream reads the pages the crawler disposed of from a key-value bucket.
package jetstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall/pagerecall"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type DisposedPages struct {
	bucket jetstream.KeyValue
}

func NewDisposedPages(bucket jetstream.KeyValue) *DisposedPages {
	return &DisposedPages{bucket: bucket}
}

func (p *DisposedPages) DisposalOf(
	ctx context.Context,
	canonicalURL string,
) (pagerecall.PageDisposal, error) {
	key := yacycrawlcontract.DisposedPageKey(canonicalURL)
	revision, err := disposalRevisionIn(ctx, p.bucket, key)
	if err != nil {
		return nil, err
	}
	return &PageDisposal{
		bucket:           p.bucket,
		key:              key,
		revisionAtLookup: revision,
	}, nil
}

func disposalRevisionIn(
	ctx context.Context,
	bucket jetstream.KeyValue,
	key string,
) (uint64, error) {
	entry, err := bucket.Get(ctx, key)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("look up disposal under %q: %w", key, err)
	}
	return entry.Revision(), nil
}

type PageDisposal struct {
	bucket           jetstream.KeyValue
	key              string
	revisionAtLookup uint64
}

func (d *PageDisposal) HasOccurred(ctx context.Context) (bool, error) {
	revision, err := disposalRevisionIn(ctx, d.bucket, d.key)
	if err != nil {
		return false, err
	}
	return revision > d.revisionAtLookup, nil
}
