package natsbroker_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

func TestStartServesJetStreamFromTheStoreDirectory(t *testing.T) {
	storeDirectory := t.TempDir()
	broker, err := natsbroker.Start(storeDirectory)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer broker.Shutdown()

	js := jetStreamOf(t, broker)
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

func jetStreamOf(t *testing.T, broker *natsbroker.Broker) jetstream.JetStream {
	t.Helper()
	js, connection, err := jetstreamconnect.Open(
		"nats://broker.internal",
		func(context.Context, string, string) (net.Conn, error) { return broker.Connection() },
		jetstreamconnect.UnobservedConnection{},
		jetstreamconnect.BrokerAccess{},
	)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(connection.Close)
	return js
}

func TestStartAcceptsAMessageOverTheDefaultNATSPayloadCeiling(t *testing.T) {
	broker, err := natsbroker.Start(t.TempDir())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer broker.Shutdown()

	js := jetStreamOf(t, broker)
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
