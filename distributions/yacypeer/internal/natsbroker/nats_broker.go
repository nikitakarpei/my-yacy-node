// Package natsbroker runs the NATS JetStream server built into the peer. It listens on no port:
// the parts of the peer connect to it inside the process.
package natsbroker

import (
	"errors"
	"fmt"
	"net"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
)

const (
	readyWait      = 10 * time.Second
	payloadCeiling = 8 << 20
)

var errNotReady = errors.New("built-in nats not ready for connections")

type Broker struct {
	server *natsserver.Server
}

func Start(storeDirectory string) (*Broker, error) {
	server, err := natsserver.NewServer(&natsserver.Options{
		DontListen: true,
		MaxPayload: payloadCeiling,
		JetStream:  true,
		StoreDir:   storeDirectory,
	})
	if err != nil {
		return nil, fmt.Errorf("built-in nats: %w", err)
	}
	server.Start()
	if !server.ReadyForConnections(readyWait) {
		server.Shutdown()
		return nil, errNotReady
	}
	return &Broker{server: server}, nil
}

func (b *Broker) Connection() (net.Conn, error) {
	connection, err := b.server.InProcessConn()
	if err != nil {
		return nil, fmt.Errorf("built-in nats: %w", err)
	}
	return connection, nil
}

func (b *Broker) Shutdown() {
	b.server.Shutdown()
	b.server.WaitForShutdown()
}
