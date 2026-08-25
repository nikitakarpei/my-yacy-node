// Package jetstream remembers, in a key-value bucket, the URL each requested URL
// redirected to when the corpus scraped it, so an exact-URL recall of the requested URL
// finds the markdown stored under the URL the origin settled on.
package jetstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const bucketName = "YACY_PAGE_MARKDOWN_REDIRECTIONS"

type PageRedirections struct {
	redirections jetstream.KeyValue
}

func OpenPageRedirections(
	ctx context.Context,
	pageMarkdownJetStream jetstream.JetStream,
) (*PageRedirections, error) {
	redirections, err := pageMarkdownJetStream.CreateOrUpdateKeyValue(
		ctx,
		jetstream.KeyValueConfig{
			Bucket:      bucketName,
			Compression: true,
			History:     1,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("ensure page redirection bucket: %w", err)
	}
	return &PageRedirections{redirections: redirections}, nil
}

func (r *PageRedirections) Record(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	markdownURL canonicalurl.CanonicalURL,
) error {
	if _, err := r.redirections.PutString(
		ctx, keyOf(requestedURL), markdownURL.String(),
	); err != nil {
		return fmt.Errorf("record redirection of %q to %q: %w", requestedURL, markdownURL, err)
	}
	return nil
}

func (r *PageRedirections) RedirectionOf(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
) (canonicalurl.CanonicalURL, bool, error) {
	entry, err := r.redirections.Get(ctx, keyOf(requestedURL))
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return canonicalurl.CanonicalURL{}, false, nil
	}
	if err != nil {
		return canonicalurl.CanonicalURL{}, false,
			fmt.Errorf("get redirection of %q: %w", requestedURL, err)
	}
	markdownURL, err := canonicalurl.CanonicalURLOf(string(entry.Value()))
	if err != nil {
		return canonicalurl.CanonicalURL{}, false,
			fmt.Errorf("read redirection of %q: %w", requestedURL, err)
	}
	return markdownURL, true, nil
}

func keyOf(requestedURL canonicalurl.CanonicalURL) string {
	sum := sha256.Sum256([]byte(requestedURL.String()))
	return hex.EncodeToString(sum[:])
}
