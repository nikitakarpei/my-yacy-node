package brokeraccess_test

import (
	"errors"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess/brokeraccesstest"
)

const (
	secret      = "pairing-secret"
	connectWait = 10 * time.Second
)

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

type startingClock struct {
	timeouts chan time.Duration
}

func (c startingClock) After(timeout time.Duration, expire func()) func() {
	c.timeouts <- timeout
	expire()
	return func() {}
}

func connect(link companionprotocol.PairLink) (*nats.Conn, error) {
	clock := startingClock{timeouts: make(chan time.Duration, len(link.Addresses))}
	return brokeraccess.NewConnector(clock).
		ConnectOnce(link, brokeraccess.PairingAccess(link, "Pixel 8"))
}

type firstAddressClock struct{}

func (firstAddressClock) After(timeout time.Duration, expire func()) func() {
	if timeout == 0 {
		expire()
	}
	return func() {}
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

	if !strings.HasSuffix(connection.ConnectedUrl(), "://"+broker.Address()) {
		t.Errorf("connected to %s, want %s", connection.ConnectedUrl(), broker.Address())
	}
}

func TestConnectStopsAtTheFirstRefusalWithoutWaitingForTheOtherAddresses(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})
	link := linkTo(broker, "another secret", broker.Address(), unansweredAddressOf(broker))
	connectErrors := make(chan error, 1)

	go func() {
		_, err := brokeraccess.NewConnector(firstAddressClock{}).
			ConnectOnce(link, brokeraccess.PairingAccess(link, "Pixel 8"))
		connectErrors <- err
	}()

	select {
	case err := <-connectErrors:
		if !errors.Is(err, nats.ErrAuthorization) {
			t.Errorf("error = %v, want an authorization error", err)
		}
	case <-time.After(connectWait):
		t.Error("Connect waits for the addresses after a refusal")
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

func TestConnectStartsEachLaterAddressOneStaggerAfterThePreviousOne(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})
	link := linkTo(broker, secret, broker.Address(), broker.Address(), broker.Address())
	clock := startingClock{timeouts: make(chan time.Duration, len(link.Addresses))}

	connection, err := brokeraccess.NewConnector(clock).
		ConnectOnce(link, brokeraccess.PairingAccess(link, "Pixel 8"))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer connection.Close()

	close(clock.timeouts)
	var timeouts []time.Duration
	for timeout := range clock.timeouts {
		timeouts = append(timeouts, timeout)
	}
	if want := []time.Duration{
		0,
		250 * time.Millisecond,
		500 * time.Millisecond,
	}; !slices.Equal(
		timeouts,
		want,
	) {
		t.Errorf("start delays = %v, want %v", timeouts, want)
	}
}
