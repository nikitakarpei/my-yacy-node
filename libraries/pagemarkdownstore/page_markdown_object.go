// Package pagemarkdownstore is the single source of truth binding a crawled
// page's canonical URL to its markdown object in the NATS JetStream Object
// Store, shared by the writer that fills the store and future readers.
package pagemarkdownstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

const BucketName = "YACY_PAGE_MARKDOWN"

func ObjectName(canonicalURL string) string {
	sum := sha256.Sum256([]byte(canonicalURL))
	return hex.EncodeToString(sum[:])
}

func EnsureBucket(ctx context.Context, js jetstream.JetStream) (jetstream.ObjectStore, error) {
	store, err := js.CreateOrUpdateObjectStore(ctx, jetstream.ObjectStoreConfig{
		Bucket: BucketName,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure page markdown bucket: %w", err)
	}
	return store, nil
}
