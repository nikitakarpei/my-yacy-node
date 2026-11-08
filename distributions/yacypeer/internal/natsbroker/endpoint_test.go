package natsbroker_test

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

const clientUser = "device-a"

type passwordAuthentication struct{}

func (passwordAuthentication) Check(client natsserver.ClientAuthentication) bool {
	if client.GetOpts().Password != "letmein" {
		return false
	}
	client.RegisterUser(&natsserver.User{Username: clientUser})
	return true
}

func startWithEndpoint(t *testing.T) *natsbroker.Broker {
	t.Helper()
	certificate, err := brokerkey.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	broker, err := natsbroker.StartWithEndpoint(t.TempDir(), natsbroker.Endpoint{
		Authentication: passwordAuthentication{},
		Address:        "127.0.0.1:0",
		Certificate:    certificate,
	})
	if err != nil {
		t.Fatalf("StartWithEndpoint: %v", err)
	}
	t.Cleanup(broker.Shutdown)
	return broker
}

func clientConnection(
	broker *natsbroker.Broker,
	options ...nats.Option,
) (*nats.Conn, error) {
	connection, err := nats.Connect(
		"tls://"+broker.EndpointAddress().String(),
		append(options, nats.MaxReconnects(0))...,
	)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return connection, nil
}

func TestAClientThatTheAuthenticationAcceptsConnectsWithTLS(t *testing.T) {
	broker := startWithEndpoint(t)

	connection, err := clientConnection(broker,
		nats.Secure(&tls.Config{InsecureSkipVerify: true}), //nolint:gosec // test broker.
		nats.UserInfo("", "letmein"),
	)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer connection.Close()

	if !broker.ConnectedUsers()[clientUser] {
		t.Errorf("connected users = %v, want %s", broker.ConnectedUsers(), clientUser)
	}
}

func TestAClientThatTheAuthenticationRefusesCannotConnect(t *testing.T) {
	broker := startWithEndpoint(t)

	connection, err := clientConnection(broker,
		nats.Secure(&tls.Config{InsecureSkipVerify: true}), //nolint:gosec // test broker.
		nats.UserInfo("", "wrong"),
	)
	if err == nil {
		connection.Close()
		t.Fatal("a refused client connected")
	}
}

func TestDisconnectUserClosesTheConnectionsOfTheUser(t *testing.T) {
	broker := startWithEndpoint(t)
	closed := make(chan struct{})
	connection, err := clientConnection(broker,
		nats.Secure(&tls.Config{InsecureSkipVerify: true}), //nolint:gosec // test broker.
		nats.UserInfo("", "letmein"),
		nats.ClosedHandler(func(*nats.Conn) { close(closed) }),
	)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer connection.Close()

	broker.DisconnectUser(clientUser)

	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("the connection of the user stays open")
	}
}

func TestThePartsOfThePeerStillConnectInsideTheProcess(t *testing.T) {
	broker := startWithEndpoint(t)

	jetStreamOf(t, broker)
}
