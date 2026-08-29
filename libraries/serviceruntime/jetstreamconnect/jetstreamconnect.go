// Package jetstreamconnect opens a NATS connection and derives its JetStream context,
// returning both, so a caller that also publishes on a core subject holds the one
// connection they share.
package jetstreamconnect

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func Open(url string) (jetstream.JetStream, *nats.Conn, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, nil, fmt.Errorf("connect nats: %w", err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("init jetstream: %w", err)
	}
	return js, conn, nil
}
