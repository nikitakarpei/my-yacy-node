package yacycrawlcontract

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

const RedirectResolutionBucketName = "YACY_REDIRECT_RESOLUTION"

type RedirectResolutionBucketSpec struct {
	MaxBytes int64
}

func RedirectResolutionKey(canonicalURL string) string {
	sum := sha256.Sum256([]byte(canonicalURL))
	return hex.EncodeToString(sum[:])
}

func EnsureRedirectResolutionBucket(
	ctx context.Context,
	js jetstream.JetStream,
	spec RedirectResolutionBucketSpec,
) error {
	if _, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   RedirectResolutionBucketName,
		MaxBytes: spec.MaxBytes,
	}); err != nil {
		return fmt.Errorf("ensure redirect resolution bucket: %w", err)
	}
	return nil
}
