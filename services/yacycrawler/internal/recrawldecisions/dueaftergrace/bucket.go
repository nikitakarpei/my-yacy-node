// Package dueaftergrace decides a page is due once a configured grace
// window has elapsed since its last visit, and revalidates it against
// stored ETag / Last-Modified validators once it is.
package dueaftergrace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const BucketName = "YACY_PAGE_VISITS"

type BucketSpec struct {
	MaxBytes  int64
	Retention time.Duration
}

func key(canonicalURL string) string {
	sum := sha256.Sum256([]byte(canonicalURL))
	return hex.EncodeToString(sum[:])
}

func Ensure(ctx context.Context, js jetstream.JetStream, spec BucketSpec) error {
	if _, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   BucketName,
		MaxBytes: spec.MaxBytes,
		TTL:      spec.Retention,
	}); err != nil {
		return fmt.Errorf("ensure page visit bucket: %w", err)
	}
	return nil
}
