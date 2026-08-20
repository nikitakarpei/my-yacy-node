// Package dueaftergrace decides a page is due once a configured grace
// window has elapsed since its last visit, and supplies the page version
// that visit recorded.
package dueaftergrace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const BucketName = "YACY_PAGE_VISITS"

type BucketSpec struct {
	MaxBytes  int64
	Retention time.Duration
}

func key(canonicalURL yacycrawlcontract.CanonicalURL) string {
	sum := sha256.Sum256([]byte(canonicalURL.String()))
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
