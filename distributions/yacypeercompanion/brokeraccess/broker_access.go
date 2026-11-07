// Package brokeraccess connects the app to the broker of a peer. It tries each address of the
// pairing link side by side, keeps the first connection that the broker accepts, and stops at the
// first refusal, since every address leads to the same broker. Its NATS options give TLS that
// accepts only the broker key that the link names, and the secret, key and inbox of the app.
package brokeraccess

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
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

func Connect(link companionprotocol.PairLink, options []nats.Option) (*nats.Conn, error) {
	if len(link.Addresses) == 0 {
		return nil, errNoAddress
	}
	attempts := make(chan attempt, len(link.Addresses))
	settled := make(chan struct{})
	for position, address := range link.Addresses {
		go func() {
			select {
			case <-time.After(time.Duration(position) * attemptStagger):
			case <-settled:
				attempts <- attempt{err: errSettled}
				return
			}
			connection, err := nats.Connect("tls://"+address, options...)
			attempts <- attempt{connection: connection, err: err}
		}()
	}
	return firstConnectionOf(attempts, len(link.Addresses), settled)
}

type attempt struct {
	connection *nats.Conn
	err        error
}

func firstConnectionOf(
	attempts <-chan attempt,
	count int,
	settled chan struct{},
) (*nats.Conn, error) {
	var lastFailure error
	for received := 1; received <= count; received++ {
		each := <-attempts
		if each.err == nil || errors.Is(each.err, nats.ErrAuthorization) {
			close(settled)
			go closeEach(attempts, count-received)
			return each.connection, wrapped(each.err)
		}
		lastFailure = each.err
	}
	return nil, wrapped(lastFailure)
}

func closeEach(attempts <-chan attempt, count int) {
	for range count {
		if each := <-attempts; each.connection != nil {
			each.connection.Close()
		}
	}
}

func wrapped(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("connect to the broker of the peer: %w", err)
}

func PairingOptions(link companionprotocol.PairLink, deviceName string) []nats.Option {
	return []nats.Option{
		nats.Secure(pinnedTLSFor(link.BrokerKeyHash, nil)),
		nats.Name(deviceName),
		nats.UserInfo("", link.Secret),
		nats.CustomInboxPrefix(companionprotocol.PairingInboxPrefixFor(link.Secret)),
		nats.NoReconnect(),
		nats.Timeout(connectTimeout),
	}
}

func DeviceOptions(link companionprotocol.PairLink, key DeviceKey) ([]nats.Option, error) {
	certificate, err := certificateOf(key)
	if err != nil {
		return nil, err
	}
	device := companionprotocol.DeviceIDOf(key.PublicKeyInfo())
	return []nats.Option{
		nats.Secure(pinnedTLSFor(link.BrokerKeyHash, []tls.Certificate{certificate})),
		nats.CustomInboxPrefix(companionprotocol.DeviceInboxPrefixOf(device)),
		nats.Timeout(connectTimeout),
	}, nil
}

func pinnedTLSFor(brokerKeyHash string, certificates []tls.Certificate) *tls.Config {
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
