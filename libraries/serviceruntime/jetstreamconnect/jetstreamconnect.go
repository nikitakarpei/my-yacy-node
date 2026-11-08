// Package jetstreamconnect opens a NATS connection through the dialer it is given and derives its
// JetStream context, returning both, so a caller that also publishes on a core subject holds the
// one connection they share. A caller can observe when the broker drops the connection and when
// it returns, and can give the TLS and the inbox prefix that the broker expects of it; closing the
// connection is no loss. Errors that the broker reports while the connection is open, such as a
// refused publish or subscription, are logged as warnings.
package jetstreamconnect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	access BrokerAccess,
) (jetstream.JetStream, *nats.Conn, error) {
	return openWith(
		url,
		dial,
		append(access.Options(),
			nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
				if err != nil {
					observer.ConnectionLost()
				}
			}),
			nats.ReconnectHandler(func(*nats.Conn) { observer.ConnectionRestored() }),
		)...,
	)
}

func openWith(
	url string,
	dial Dial,
	options ...nats.Option,
) (jetstream.JetStream, *nats.Conn, error) {
	conn, err := nats.Connect(
		url,
		append(options,
			nats.SetCustomDialer(timedDialer{dial: dial}),
			nats.SkipHostLookup(),
			nats.ErrorHandler(reportBrokerError),
		)...,
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

func reportBrokerError(_ *nats.Conn, subscription *nats.Subscription, err error) {
	attributes := []any{slog.Any("error", err)}
	if subscription != nil {
		attributes = append(attributes, slog.String("subject", subscription.Subject))
	}
	slog.WarnContext(context.Background(), "the broker reported an error", attributes...)
}
