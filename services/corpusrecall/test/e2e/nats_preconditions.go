//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	natsStateAppearanceWait = 60 * time.Second
)

func connectJetStream(t *testing.T, url string) jetstream.JetStream {
	t.Helper()
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	return js
}

func awaitRecallPreconditions(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	awaitNATSState(t, "orders stream", func() bool {
		_, err := js.Stream(ctx, yacycrawlcontract.OrdersStreamName)
		return err == nil
	})
	awaitNATSState(t, "redirect resolution bucket", func() bool {
		_, err := js.KeyValue(ctx, yacycrawlcontract.RedirectResolutionBucketName)
		return err == nil
	})
	awaitNATSState(t, "disposed pages bucket", func() bool {
		_, err := js.KeyValue(ctx, yacycrawlcontract.DisposedPagesBucketName)
		return err == nil
	})
	awaitNATSState(t, "page markdown bucket", func() bool {
		_, err := js.ObjectStore(ctx, pagemarkdownstore.BucketName)
		return err == nil
	})
}

func awaitNATSState(t *testing.T, state string, exists func() bool) {
	t.Helper()
	if !pollwait.For(natsStateAppearanceWait, exists) {
		t.Fatalf("%s did not appear within %s", state, natsStateAppearanceWait)
	}
}
