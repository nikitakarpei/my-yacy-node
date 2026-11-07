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
	host string,
	name string,
	linkSecret string,
) companionprotocol.PairLink {
	_, port, _ := net.SplitHostPort(broker.Address())
	return companionprotocol.PairLink{
		Name:          name,
		Address:       net.JoinHostPort(host, port),
		BrokerKeyHash: broker.KeyHash,
		Secret:        linkSecret,
	}
}

func TestConnectReachesTheBrokerByNameWhenTheAddressDoesNotAnswer(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})
	link := linkTo(broker, "127.0.0.2", "localhost", secret)

	connection, err := brokeraccess.Connect(link, brokeraccess.PairingOptions(link, "Pixel 8"))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	connection.Close()
}

func TestConnectReportsARefusedSecretAlthoughTheLinkNamesAHost(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})
	link := linkTo(broker, "127.0.0.1", "nowhere.invalid", "another secret")

	_, err := brokeraccess.Connect(link, brokeraccess.PairingOptions(link, "Pixel 8"))

	if !errors.Is(err, nats.ErrAuthorization) {
		t.Errorf("error = %v, want an authorization error", err)
	}
}

func TestConnectRefusesABrokerWithAnotherKey(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})
	link := linkTo(broker, "127.0.0.1", "", secret)
	link.BrokerKeyHash = "another key"

	if _, err := brokeraccess.Connect(
		link,
		brokeraccess.PairingOptions(link, "Pixel 8"),
	); err == nil {
		t.Error("a broker with another key accepted the connection")
	}
}
