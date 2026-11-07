// Package brokeraccess builds the NATS options with which the app connects to the broker of a
// peer: TLS that accepts only the broker key that the pairing link names, the secret and inbox of
// a pairing, and the key and inbox of a paired device.
package brokeraccess

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

const connectTimeout = 10 * time.Second

var errBrokerKeyMismatch = errors.New("the broker key is not the key of the pairing link")

func Connect(link companionprotocol.PairLink, options []nats.Option) (*nats.Conn, error) {
	connection, err := nats.Connect("tls://"+link.Address, options...)
	_, port, splitErr := net.SplitHostPort(link.Address)
	if err == nil || errors.Is(err, nats.ErrAuthorization) || link.Name == "" || splitErr != nil {
		return connection, wrapped(err)
	}
	connection, err = nats.Connect("tls://"+net.JoinHostPort(link.Name, port), options...)
	return connection, wrapped(err)
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
