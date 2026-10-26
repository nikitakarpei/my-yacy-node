package natsbroker_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

func TestStartServesJetStreamFromTheStoreDirectory(t *testing.T) {
	storeDirectory := t.TempDir()
	broker, err := natsbroker.Start(storeDirectory)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer broker.Shutdown()

	js := natstestserver.ConnectJetStream(t, broker.ClientURL())
	if _, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "PAGES",
		Subjects: []string{"page"},
	}); err != nil {
		t.Fatalf("create stream: %v", err)
	}
	if _, err := os.Stat(filepath.Join(storeDirectory, "jetstream")); err != nil {
		t.Errorf("store directory holds no jetstream state: %v", err)
	}
}

func TestStartAcceptsAMessageOverTheDefaultNATSPayloadCeiling(t *testing.T) {
	broker, err := natsbroker.Start(t.TempDir())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer broker.Shutdown()

	js := natstestserver.ConnectJetStream(t, broker.ClientURL())
	if _, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "PAGES",
		Subjects: []string{"page"},
	}); err != nil {
		t.Fatalf("create stream: %v", err)
	}
	const pageOfferBytes = 2 << 20
	if _, err := js.Publish(
		context.Background(),
		"page",
		make([]byte, pageOfferBytes),
	); err != nil {
		t.Errorf("publish a page offer of %d bytes: %v", pageOfferBytes, err)
	}
}
