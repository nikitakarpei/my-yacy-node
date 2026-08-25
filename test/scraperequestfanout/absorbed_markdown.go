//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

const absorptionDeadline = 90 * time.Second

func absorbedMarkdownOf(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	pageURL string,
) string {
	t.Helper()
	objectName := pagemarkdownstore.ObjectNameOf(canonicalurltest.CanonicalURLOf(t, pageURL))

	var store jetstream.ObjectStore
	opened := pollwait.For(absorptionDeadline, func() bool {
		found, err := js.ObjectStore(ctx, pagemarkdownstore.BucketName)
		if err != nil {
			return false
		}
		store = found
		return true
	})
	if !opened {
		t.Fatalf("corpusmarkdown never opened the %s bucket within %s",
			pagemarkdownstore.BucketName, absorptionDeadline)
	}

	var markdown []byte
	stored := pollwait.For(absorptionDeadline, func() bool {
		found, err := store.GetBytes(ctx, objectName)
		if err != nil {
			return false
		}
		markdown = found
		return true
	})
	if !stored {
		t.Fatalf("corpusmarkdown never stored %q within %s", objectName, absorptionDeadline)
	}
	return string(markdown)
}
