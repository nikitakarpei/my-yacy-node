// Package brokeraccess connects the app to the broker of a peer. It tries each address of the
// pairing link side by side, keeps the first connection that the broker accepts, and stops at the
// first refusal, since every address leads to the same broker. Its broker access gives TLS that
// accepts only the broker key that the link names, and the secret, key and inbox of the app.
// Errors that the broker reports while a connection is open, such as a refused publish or
// subscription, are logged as warnings.
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
	errBrokerKeyMismatch = errors.New("the broker key is not the key of the pairing link")
	errNoAddress         = errors.New("the pairing link names no address")
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

func (c Connector) Connect(
	link companionprotocol.PairLink,
	options []nats.Option,
) (*nats.Conn, error) {
	if len(link.Addresses) == 0 {
		return nil, errNoAddress
	}
	connectOptions := append(
		slices.Clip(options),
		nats.Timeout(connectTimeout),
		nats.ErrorHandler(reportBrokerError),
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
	connection, err := nats.Connect("tls://"+address, options...)
	return attempt{connection: connection, err: err}
}

func reportBrokerError(_ *nats.Conn, subscription *nats.Subscription, err error) {
	attributes := []any{slog.Any("error", err)}
	if subscription != nil {
		attributes = append(attributes, slog.String("subject", subscription.Subject))
	}
	slog.WarnContext(
		context.Background(),
		"the broker of the peer reported an error",
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
	var lastFailure error
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
	return fmt.Errorf("connect to the broker of the peer: %w", err)
}

func PairingOptions(
	link companionprotocol.PairLink,
	deviceName companionprotocol.DeviceName,
) []nats.Option {
	return append(jetstreamconnect.BrokerAccess{
		TLS:         pinnedTLSFor(link.BrokerKeyHash, nil),
		InboxPrefix: companionprotocol.PairingInboxPrefixFor(link.Secret),
	}.Options(),
		nats.Name(string(deviceName)),
		nats.UserInfo("", string(link.Secret)),
		nats.NoReconnect(),
	)
}

func DeviceAccess(
	link companionprotocol.PairLink,
	key DeviceKey,
) (jetstreamconnect.BrokerAccess, error) {
	certificate, err := certificateOf(key)
	if err != nil {
		return jetstreamconnect.BrokerAccess{}, err
	}
	device := companionprotocol.DeviceIDOf(key.PublicKeyInfo())
	return jetstreamconnect.BrokerAccess{
		TLS:         pinnedTLSFor(link.BrokerKeyHash, []tls.Certificate{certificate}),
		InboxPrefix: companionprotocol.DeviceInboxPrefixOf(device),
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
			if len(state.PeerCertificates) == 0 || companionprotocol.BrokerKeyHashOf(
				state.PeerCertificates[0].RawSubjectPublicKeyInfo,
			) != brokerKeyHash {
				return errBrokerKeyMismatch
			}
			return nil
		},
	}
}
