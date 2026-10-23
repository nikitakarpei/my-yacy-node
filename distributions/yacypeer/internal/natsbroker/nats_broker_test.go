package natsbroker_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

func TestStartServesJetStreamFromTheStoreDirectory(t *testing.T) {
	storeDirectory := t.TempDir()
	broker, err := natsbroker.Start(storeDirectory)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer broker.Shutdown()

	connection, err := nats.Connect(broker.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer connection.Close()
	js, err := jetstream.New(connection)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
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
