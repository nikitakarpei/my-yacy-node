// Package jetstream holds each crawled page's markdown in a JetStream object store bucket.
package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

type Corpus struct {
	objects jetstream.ObjectStore
}

func OpenCorpus(
	ctx context.Context,
	pageMarkdownJetStream jetstream.JetStream,
) (*Corpus, error) {
	objects, err := pageMarkdownJetStream.CreateOrUpdateObjectStore(
		ctx,
		jetstream.ObjectStoreConfig{
			Bucket:      pagemarkdownstore.BucketName,
			Compression: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("ensure page markdown bucket: %w", err)
	}
	return &Corpus{objects: objects}, nil
}

func (c *Corpus) Put(ctx context.Context, canonicalURL string, markdown []byte) error {
	objectName := pagemarkdownstore.ObjectName(canonicalURL)
	if _, err := c.objects.PutBytes(ctx, objectName, markdown); err != nil {
		return fmt.Errorf("put markdown for %q: %w", canonicalURL, err)
	}
	return nil
}
