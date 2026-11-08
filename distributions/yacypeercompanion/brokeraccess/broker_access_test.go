package brokeraccess_test

import (
	"errors"
	"net"
	"testing"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess/brokeraccesstest"
)

const secret = "pairing-secret"

type secretCheck struct{}

func (secretCheck) Check(client natsserver.ClientAuthentication) bool {
	return client.GetOpts().Password == secret
}

func linkTo(
	broker brokeraccesstest.Broker,
	linkSecret companionprotocol.Secret,
	addresses ...string,
) companionprotocol.PairLink {
	return companionprotocol.PairLink{
		Addresses: addresses, BrokerKeyHash: broker.KeyHash, Secret: linkSecret,
	}
}

func unansweredAddressOf(broker brokeraccesstest.Broker) string {
	_, port, _ := net.SplitHostPort(broker.Address())
	return net.JoinHostPort("127.0.0.2", port)
}

func connect(link companionprotocol.PairLink) (*nats.Conn, error) {
	return brokeraccess.Connect(link, brokeraccess.PairingOptions(link, "Pixel 8"))
}

func TestConnectReachesTheBrokerAtALaterAddressWhenAnEarlierOneDoesNotAnswer(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})

	connection, err := connect(
		linkTo(broker, secret, unansweredAddressOf(broker), broker.Address()),
	)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	connection.Close()
}

func TestConnectPassesOverAnAddressWhereAnotherBrokerAnswers(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})
	stranger := brokeraccesstest.Start(t, secretCheck{})

	connection, err := connect(linkTo(broker, secret, stranger.Address(), broker.Address()))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer connection.Close()

	if connection.ConnectedUrl() != "tls://"+broker.Address() {
		t.Errorf("connected to %s, want %s", connection.ConnectedUrl(), broker.Address())
	}
}

func TestConnectReportsARefusedSecretAlthoughOtherAddressesRemain(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})

	_, err := connect(
		linkTo(broker, "another secret", broker.Address(), unansweredAddressOf(broker)),
	)

	if !errors.Is(err, nats.ErrAuthorization) {
		t.Errorf("error = %v, want an authorization error", err)
	}
}

func TestConnectFailsWhenNoAddressHasTheBrokerOfTheLink(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})
	stranger := brokeraccesstest.Start(t, secretCheck{})

	if _, err := connect(
		linkTo(broker, secret, stranger.Address(), unansweredAddressOf(broker)),
	); err == nil {
		t.Error("a link without the broker connected")
	}
	if _, err := connect(linkTo(broker, secret)); err == nil {
		t.Error("a link without addresses connected")
	}
}
