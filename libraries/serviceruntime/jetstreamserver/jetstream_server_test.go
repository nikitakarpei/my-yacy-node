package jetstreamserver_test

import (
	"bytes"
	"context"
	"net"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamserver"
)

const (
	streamName    = "PAGES"
	streamSubject = "page"
)

func startedServer(t *testing.T, config jetstreamserver.Config) *jetstreamserver.Server {
	t.Helper()
	server, err := jetstreamserver.Start(config)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	return server
}

func brokerOf(t *testing.T, server *jetstreamserver.Server) jetstream.JetStream {
	t.Helper()
	broker, connection, err := jetstreamconnect.Open(server.ClientURL())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(connection.Close)
	return broker
}

func pageStreamOf(t *testing.T, broker jetstream.JetStream) jetstream.Stream {
	t.Helper()
	stream, err := broker.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{streamSubject},
	})
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}
	return stream
}

func publishPage(t *testing.T, broker jetstream.JetStream, page []byte) {
	t.Helper()
	if _, err := broker.Publish(context.Background(), streamSubject, page); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

func messagesIn(t *testing.T, stream jetstream.Stream) uint64 {
	t.Helper()
	info, err := stream.Info(context.Background())
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	return info.State.Msgs
}

func TestStartServesJetStreamOnItsClientURL(t *testing.T) {
	server := startedServer(t, jetstreamserver.Config{Addr: ":0", StoreDir: t.TempDir()})
	t.Cleanup(server.Shutdown)
	broker := brokerOf(t, server)
	stream := pageStreamOf(t, broker)

	publishPage(t, broker, []byte("page"))

	if messages := messagesIn(t, stream); messages != 1 {
		t.Fatalf("messages = %d, want 1", messages)
	}
}

func TestStartCarriesAPageOfFourMegabytes(t *testing.T) {
	server := startedServer(t, jetstreamserver.Config{Addr: "127.0.0.1:0", StoreDir: t.TempDir()})
	t.Cleanup(server.Shutdown)
	broker := brokerOf(t, server)
	stream := pageStreamOf(t, broker)

	publishPage(t, broker, bytes.Repeat([]byte("p"), 4<<20))

	if messages := messagesIn(t, stream); messages != 1 {
		t.Fatalf("messages = %d, want 1", messages)
	}
}

func TestStartKeepsStreamsInTheStoreDirAcrossARestart(t *testing.T) {
	config := jetstreamserver.Config{Addr: "127.0.0.1:0", StoreDir: t.TempDir()}
	first := startedServer(t, config)
	firstBroker := brokerOf(t, first)
	pageStreamOf(t, firstBroker)
	publishPage(t, firstBroker, []byte("page"))
	first.Shutdown()

	second := startedServer(t, config)
	t.Cleanup(second.Shutdown)
	stream, err := brokerOf(t, second).Stream(context.Background(), streamName)
	if err != nil {
		t.Fatalf("open stream after restart: %v", err)
	}

	if messages := messagesIn(t, stream); messages != 1 {
		t.Fatalf("messages after restart = %d, want 1", messages)
	}
}

func TestStartRejectsAnAddressWithoutAPort(t *testing.T) {
	if _, err := jetstreamserver.Start(jetstreamserver.Config{
		Addr: "127.0.0.1", StoreDir: t.TempDir(),
	}); err == nil {
		t.Fatal("expected an error for an address without a port")
	}
}

func TestStartRejectsAnAddressWithANamedPort(t *testing.T) {
	if _, err := jetstreamserver.Start(jetstreamserver.Config{
		Addr: "127.0.0.1:nats", StoreDir: t.TempDir(),
	}); err == nil {
		t.Fatal("expected an error for a port that is not a number")
	}
}

func TestStartFailsOnAnAddressAnotherListenerHolds(t *testing.T) {
	t.Parallel()
	listener, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	if _, err := jetstreamserver.Start(jetstreamserver.Config{
		Addr: listener.Addr().String(), StoreDir: t.TempDir(),
	}); err == nil {
		t.Fatal("expected an error for an address another listener holds")
	}
}
