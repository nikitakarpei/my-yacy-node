// Package brokeraccess connects the app to the broker of a peer. It tries each address of the
// pair link side by side, keeps the first connection that the broker accepts, and stops at the
// first refusal, since every address leads to the same broker. It connects once or with
// reconnects, with the access of a pairing device or of a paired device, both with TLS that accepts
// only the broker key of the link. Errors on an open connection, such as a refused publish, are
// reported.
package brokeraccess

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

const (
	connectTimeout = 10 * time.Second
	attemptStagger = 250 * time.Millisecond
)

var (
	ErrNoConnection = errors.New("no connection to the broker of the peer")

	errBrokerKeyMismatch = errors.New("the broker key is not the key of the pair link")
	errNoAddress         = errors.New("the pair link names no address")
	errSettled           = errors.New("another address answered first")
)

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type Connector struct {
	clock Clock
}

func NewConnector(clock Clock) Connector {
	return Connector{clock: clock}
}

func (c Connector) ConnectOnce(
	link companionprotocol.PairLink,
	access jetstreamconnect.BrokerAccess,
) (*nats.Conn, error) {
	return c.connect(link, append(access.Options(), nats.NoReconnect()))
}

func (c Connector) ConnectWithReconnects(
	link companionprotocol.PairLink,
	access jetstreamconnect.BrokerAccess,
	onRefused func(),
) (*nats.Conn, error) {
	return c.connect(link, append(access.Options(), nats.ClosedHandler(func(closed *nats.Conn) {
		if errors.Is(closed.LastError(), nats.ErrAuthorization) {
			onRefused()
		}
	})))
}

func (c Connector) connect(
	link companionprotocol.PairLink,
	options []nats.Option,
) (*nats.Conn, error) {
	connectOptions := append(
		slices.Clip(options),
		nats.Timeout(connectTimeout),
		nats.ErrorHandler(reportConnectionError),
	)
	attempts := make(chan attempt, len(link.Addresses))
	settled := make(chan struct{})
	for position, address := range link.Addresses {
		c.clock.After(time.Duration(position)*attemptStagger, func() {
			go func() { attempts <- attemptAt(address, connectOptions, settled) }()
		})
	}
	return awaitFirstConnection(attempts, len(link.Addresses), settled)
}

func attemptAt(address string, options []nats.Option, settled <-chan struct{}) attempt {
	select {
	case <-settled:
		return attempt{err: errSettled}
	default:
	}
	connection, err := nats.Connect(brokerURLOf(address), options...)
	return attempt{connection: connection, err: err}
}

func brokerURLOf(address string) string {
	return "tls://" + address
}

func reportConnectionError(_ *nats.Conn, subscription *nats.Subscription, err error) {
	attributes := []any{slog.Any("error", err)}
	if subscription != nil {
		attributes = append(attributes, slog.String("subject", subscription.Subject))
	}
	slog.WarnContext(
		context.Background(),
		"the connection to the broker of the peer reported an error",
		attributes...)
}

type attempt struct {
	connection *nats.Conn
	err        error
}

func awaitFirstConnection(
	attempts <-chan attempt,
	count int,
	settled chan struct{},
) (*nats.Conn, error) {
	lastFailure := errNoAddress
	for received := 1; received <= count; received++ {
		result := <-attempts
		if result.err == nil || errors.Is(result.err, nats.ErrAuthorization) {
			close(settled)
			go closeLateConnections(attempts, count-received)
			return result.connection, connectFailureFrom(result.err)
		}
		lastFailure = result.err
	}
	return nil, connectFailureFrom(lastFailure)
}

func closeLateConnections(attempts <-chan attempt, count int) {
	for range count {
		if result := <-attempts; result.connection != nil {
			result.connection.Close()
		}
	}
}

func connectFailureFrom(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrNoConnection, err)
}

func PairingAccess(
	link companionprotocol.PairLink,
	deviceName companionprotocol.DeviceName,
) jetstreamconnect.BrokerAccess {
	credentials := companionprotocol.PairingCredentialsFrom(deviceName, link.Secret)
	return jetstreamconnect.BrokerAccess{
		TLS:         pinnedTLSFor(link.BrokerKeyHash, nil),
		ClientName:  credentials.ClientName,
		Password:    credentials.Password,
		InboxPrefix: companionprotocol.PairingInboxPrefixFor(link.Secret),
	}
}

func DeviceAccess(
	link companionprotocol.PairLink,
	key DeviceKey,
) (jetstreamconnect.BrokerAccess, error) {
	certificate, err := certificateOf(key)
	if err != nil {
		return jetstreamconnect.BrokerAccess{}, err
	}
	deviceID := companionprotocol.DeviceIDFrom(key.PublicKeyInfo())
	return jetstreamconnect.BrokerAccess{
		TLS:         pinnedTLSFor(link.BrokerKeyHash, []tls.Certificate{certificate}),
		InboxPrefix: companionprotocol.DeviceInboxPrefixOf(deviceID),
	}, nil
}

func pinnedTLSFor(
	brokerKeyHash companionprotocol.BrokerKeyHash,
	certificates []tls.Certificate,
) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // VerifyConnection pins the broker key.
		MinVersion:         tls.VersionTLS13,
		Certificates:       certificates,
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 || companionprotocol.BrokerKeyHashFrom(
				state.PeerCertificates[0].RawSubjectPublicKeyInfo,
			) != brokerKeyHash {
				return errBrokerKeyMismatch
			}
			return nil
		},
	}
}
