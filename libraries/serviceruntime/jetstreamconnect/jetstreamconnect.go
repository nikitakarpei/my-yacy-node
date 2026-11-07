// Package jetstreamconnect opens a NATS connection through the dialer it is given and derives its
// JetStream context, returning both, so a caller that also publishes on a core subject holds the
// one connection they share. A caller can observe when the broker drops the connection and when
// it returns; closing the connection is no loss.
package jetstreamconnect

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var ErrBrokerUnreachable = errors.New("connect nats")

type Dial func(ctx context.Context, network, address string) (net.Conn, error)

type ConnectionObserver interface {
	ConnectionLost()
	ConnectionRestored()
}

func Open(url string, dial Dial) (jetstream.JetStream, *nats.Conn, error) {
	return openWith(url, dial)
}

func OpenObserved(
	url string,
	dial Dial,
	observer ConnectionObserver,
) (jetstream.JetStream, *nats.Conn, error) {
	return openWith(
		url,
		dial,
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				observer.ConnectionLost()
			}
		}),
		nats.ReconnectHandler(func(*nats.Conn) { observer.ConnectionRestored() }),
	)
}

func openWith(
	url string,
	dial Dial,
	options ...nats.Option,
) (jetstream.JetStream, *nats.Conn, error) {
	conn, err := nats.Connect(
		url,
		append(options, nats.SetCustomDialer(timedDialer{dial: dial}), nats.SkipHostLookup())...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrBrokerUnreachable, err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("init jetstream: %w", err)
	}
	return js, conn, nil
}

type timedDialer struct {
	dial Dial
}

func (d timedDialer) Dial(network, address string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), nats.GetDefaultOptions().Timeout)
	defer cancel()
	return d.dial(ctx, network, address)
}
