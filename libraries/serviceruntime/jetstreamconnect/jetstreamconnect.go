// Package jetstreamconnect opens a NATS connection and derives its JetStream context,
// returning both, so a caller that also publishes on a core subject holds the one
// connection they share. A caller can observe when the broker drops the connection and when
// it returns; closing the connection is no loss.
package jetstreamconnect

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var ErrBrokerUnreachable = errors.New("connect nats")

type ConnectionObserver interface {
	ConnectionLost()
	ConnectionRestored()
}

func Open(url string) (jetstream.JetStream, *nats.Conn, error) {
	return openWith(url)
}

func OpenObserved(
	url string,
	observer ConnectionObserver,
) (jetstream.JetStream, *nats.Conn, error) {
	return openWith(
		url,
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				observer.ConnectionLost()
			}
		}),
		nats.ReconnectHandler(func(*nats.Conn) { observer.ConnectionRestored() }),
	)
}

func openWith(url string, options ...nats.Option) (jetstream.JetStream, *nats.Conn, error) {
	conn, err := nats.Connect(url, options...)
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
