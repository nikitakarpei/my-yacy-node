package yacycrawlcontract

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const DisposedPagesBucketName = "YACY_DISPOSED_PAGES"

type DisposedPagesBucketSpec struct {
	MaxBytes  int64
	Retention time.Duration
}

func DisposedPageKey(canonicalURL string) string {
	sum := sha256.Sum256([]byte(canonicalURL))
	return hex.EncodeToString(sum[:])
}

func EnsureDisposedPagesBucket(
	ctx context.Context,
	js jetstream.JetStream,
	spec DisposedPagesBucketSpec,
) error {
	if _, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   DisposedPagesBucketName,
		MaxBytes: spec.MaxBytes,
		TTL:      spec.Retention,
	}); err != nil {
		return fmt.Errorf("ensure disposed pages bucket: %w", err)
	}
	return nil
}
