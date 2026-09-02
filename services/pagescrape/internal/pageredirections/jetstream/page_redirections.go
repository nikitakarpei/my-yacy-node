package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const redirectionsKeptPerURL = 1

type PageRedirections struct {
	redirections jetstream.KeyValue
}

func CreatePageRedirections(
	ctx context.Context,
	broker jetstream.JetStream,
) (*PageRedirections, error) {
	redirections, err := broker.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      pagescrapecontract.PageRedirectionsBucketName,
		Compression: true,
		History:     redirectionsKeptPerURL,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"create the %s bucket: %w", pagescrapecontract.PageRedirectionsBucketName, err,
		)
	}
	return &PageRedirections{redirections: redirections}, nil
}

func (r *PageRedirections) Record(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	pageURL canonicalurl.CanonicalURL,
) error {
	key := pagescrapecontract.PageRedirectionKeyOf(requestedURL)
	if _, err := r.redirections.PutString(ctx, key, pageURL.String()); err != nil {
		return fmt.Errorf("record the redirection of %q to %q: %w", requestedURL, pageURL, err)
	}
	return nil
}
