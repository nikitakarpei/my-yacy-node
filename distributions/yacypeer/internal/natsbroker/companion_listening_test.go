package natsbroker_test

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

const companionUser = "device-a"

type passwordAuthentication struct{}

func (passwordAuthentication) Check(client natsserver.ClientAuthentication) bool {
	if client.GetTLSConnectionState() == nil {
		return true
	}
	if client.GetOpts().Password != "letmein" {
		return false
	}
	client.RegisterUser(&natsserver.User{Username: companionUser})
	return true
}

func startForCompanions(t *testing.T) *natsbroker.Broker {
	t.Helper()
	certificate, err := brokerkey.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	broker, err := natsbroker.StartForCompanions(t.TempDir(), natsbroker.CompanionListening{
		Authentication: passwordAuthentication{},
		Address:        "127.0.0.1:0",
		Certificate:    certificate,
	})
	if err != nil {
		t.Fatalf("StartForCompanions: %v", err)
	}
	t.Cleanup(broker.Shutdown)
	return broker
}

func companionConnection(
	broker *natsbroker.Broker,
	options ...nats.Option,
) (*nats.Conn, error) {
	connection, err := nats.Connect(
		"tls://"+broker.CompanionAddress().String(),
		append(options, nats.MaxReconnects(0))...,
	)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return connection, nil
}

func TestACompanionThatTheAuthenticationAcceptsConnectsWithTLS(t *testing.T) {
	broker := startForCompanions(t)

	connection, err := companionConnection(broker,
		nats.Secure(&tls.Config{InsecureSkipVerify: true}), //nolint:gosec // test broker.
		nats.UserInfo("", "letmein"),
	)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer connection.Close()

	if !broker.ConnectedUsers()[companionUser] {
		t.Errorf("connected users = %v, want %s", broker.ConnectedUsers(), companionUser)
	}
}

func TestACompanionThatTheAuthenticationRefusesCannotConnect(t *testing.T) {
	broker := startForCompanions(t)

	connection, err := companionConnection(broker,
		nats.Secure(&tls.Config{InsecureSkipVerify: true}), //nolint:gosec // test broker.
		nats.UserInfo("", "wrong"),
	)
	if err == nil {
		connection.Close()
		t.Fatal("a refused companion connected")
	}
}

func TestDisconnectUserClosesTheConnectionsOfTheUser(t *testing.T) {
	broker := startForCompanions(t)
	closed := make(chan struct{})
	connection, err := companionConnection(broker,
		nats.Secure(&tls.Config{InsecureSkipVerify: true}), //nolint:gosec // test broker.
		nats.UserInfo("", "letmein"),
		nats.ClosedHandler(func(*nats.Conn) { close(closed) }),
	)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer connection.Close()

	broker.DisconnectUser(companionUser)

	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("the connection of the user stays open")
	}
}

func TestThePartsOfThePeerStillConnectInsideTheProcess(t *testing.T) {
	broker := startForCompanions(t)

	jetStreamOf(t, broker)
}
