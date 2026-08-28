package jetstreamrecord

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type BucketSpec struct {
	MaxBytes  int64
	Retention time.Duration
}

func EnsureBucket(
	ctx context.Context,
	js jetstream.JetStream,
	name string,
	spec BucketSpec,
) error {
	if _, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   name,
		MaxBytes: spec.MaxBytes,
		TTL:      spec.Retention,
	}); err != nil {
		return fmt.Errorf("ensure bucket %s: %w", name, err)
	}
	return nil
}
