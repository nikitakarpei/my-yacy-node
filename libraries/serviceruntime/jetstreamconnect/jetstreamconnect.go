// Package jetstreamconnect opens a NATS connection and its JetStream context,
// returning the JetStream handle alongside a closer for the connection.
package jetstreamconnect

import (
	"fmt"
	"io"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func Open(url string) (jetstream.JetStream, io.Closer, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, nil, fmt.Errorf("connect nats: %w", err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("init jetstream: %w", err)
	}
	return js, connectionCloser{conn}, nil
}

type connectionCloser struct {
	conn *nats.Conn
}

func (c connectionCloser) Close() error {
	c.conn.Close()
	return nil
}
