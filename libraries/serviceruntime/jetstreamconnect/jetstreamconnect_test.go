package jetstreamconnect_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

const callbackWait = 5 * time.Second

func TestOpenConnectsAndCloses(t *testing.T) {
	url := natstestserver.Start(t)
	js, connection, err := jetstreamconnect.Open(url, (&net.Dialer{}).DialContext)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if js == nil {
		t.Fatal("Open returned nil jetstream")
	}
	connection.Close()
}

func TestOpenConnectsThroughTheDialer(t *testing.T) {
	broker, err := natsserver.NewServer(
		&natsserver.Options{DontListen: true, JetStream: true, StoreDir: t.TempDir()},
	)
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	go broker.Start()
	if !broker.ReadyForConnections(callbackWait) {
		t.Fatal("broker not ready")
	}
	t.Cleanup(broker.Shutdown)

	js, connection, err := jetstreamconnect.Open(
		"nats://broker.internal:4222",
		func(context.Context, string, string) (net.Conn, error) { return broker.InProcessConn() },
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer connection.Close()
	if _, err := js.AccountInfo(context.Background()); err != nil {
		t.Errorf("account info through the dialer: %v", err)
	}
}

func TestOpenRejectsUnreachable(t *testing.T) {
	_, _, err := jetstreamconnect.Open("nats://127.0.0.1:1", (&net.Dialer{}).DialContext)
	if !errors.Is(err, jetstreamconnect.ErrBrokerUnreachable) {
		t.Fatalf("err = %v, want ErrBrokerUnreachable", err)
	}
}

type connectionEvents chan string

func (events connectionEvents) ConnectionLost()     { events <- "lost" }
func (events connectionEvents) ConnectionRestored() { events <- "restored" }

func brokerOn(t *testing.T, port int) *natsserver.Server {
	t.Helper()
	broker, err := natsserver.NewServer(&natsserver.Options{Host: "127.0.0.1", Port: port})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	go broker.Start()
	if !broker.ReadyForConnections(callbackWait) {
		t.Fatal("broker not ready")
	}
	t.Cleanup(broker.Shutdown)
	return broker
}

func awaitEvent(t *testing.T, events connectionEvents, want string) {
	t.Helper()
	select {
	case event := <-events:
		if event != want {
			t.Fatalf("event = %q, want %q", event, want)
		}
	case <-time.After(callbackWait):
		t.Fatalf("no %s event", want)
	}
}

func TestOpenObservedReportsALostAndARestoredConnection(t *testing.T) {
	broker := brokerOn(t, -1)
	port := broker.Addr().(*net.TCPAddr).Port
	events := make(connectionEvents, 2)
	_, connection, err := jetstreamconnect.OpenObserved(
		broker.ClientURL(),
		(&net.Dialer{}).DialContext,
		events,
	)
	if err != nil {
		t.Fatalf("OpenObserved: %v", err)
	}
	defer connection.Close()

	broker.Shutdown()
	awaitEvent(t, events, "lost")
	brokerOn(t, port)
	awaitEvent(t, events, "restored")
}

func TestOpenObservedRejectsUnreachable(t *testing.T) {
	_, _, err := jetstreamconnect.OpenObserved(
		"nats://127.0.0.1:1",
		(&net.Dialer{}).DialContext,
		make(connectionEvents, 1),
	)
	if !errors.Is(err, jetstreamconnect.ErrBrokerUnreachable) {
		t.Fatalf("err = %v, want ErrBrokerUnreachable", err)
	}
}

func TestOpenObservedConnectsWithTheOptionsOfTheCaller(t *testing.T) {
	broker := brokerOn(t, -1)
	_, connection, err := jetstreamconnect.OpenObserved(
		broker.ClientURL(),
		(&net.Dialer{}).DialContext,
		make(connectionEvents, 1),
		nats.Name("companion"),
	)
	if err != nil {
		t.Fatalf("OpenObserved: %v", err)
	}
	defer connection.Close()

	connections, err := broker.Connz(&natsserver.ConnzOptions{})
	if err != nil {
		t.Fatalf("Connz: %v", err)
	}
	if len(connections.Conns) != 1 || connections.Conns[0].Name != "companion" {
		t.Errorf("connections = %+v, want one named companion", connections.Conns)
	}
}
