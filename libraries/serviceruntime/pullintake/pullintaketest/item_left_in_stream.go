package pullintaketest

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const countPollPause = 50 * time.Millisecond

func RequireItemLeftInStream(
	t *testing.T,
	ctx context.Context,
	consumer jetstream.Consumer,
	within time.Duration,
) {
	t.Helper()
	info := infoOnceCounted(t, ctx, consumer, within)
	if info.NumPending != 1 || info.NumAckPending != 0 {
		t.Errorf("pending = %d, taken = %d, want the item left in the stream",
			info.NumPending, info.NumAckPending)
	}
}

func infoOnceCounted(
	t *testing.T,
	ctx context.Context,
	consumer jetstream.Consumer,
	within time.Duration,
) *jetstream.ConsumerInfo {
	t.Helper()
	deadline := time.Now().Add(within)
	for {
		info, err := consumer.Info(ctx)
		if err != nil {
			t.Fatalf("consumer info: %v", err)
		}
		if info.NumPending > 0 || info.NumAckPending > 0 || !time.Now().Before(deadline) {
			return info
		}
		time.Sleep(countPollPause)
	}
}
