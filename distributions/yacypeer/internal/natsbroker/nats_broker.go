// Package natsbroker runs the NATS JetStream server built into the peer, listening on the
// loopback interface only.
package natsbroker

import (
	"errors"
	"fmt"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
)

const (
	loopbackHost = "127.0.0.1"
	anyFreePort  = -1
	readyWait    = 10 * time.Second
)

var errNotReady = errors.New("built-in nats not ready for connections")

type Broker struct {
	server *natsserver.Server
}

func Start(storeDirectory string) (*Broker, error) {
	server, err := natsserver.NewServer(&natsserver.Options{
		Host:      loopbackHost,
		Port:      anyFreePort,
		JetStream: true,
		StoreDir:  storeDirectory,
	})
	if err != nil {
		return nil, fmt.Errorf("built-in nats: %w", err)
	}
	go server.Start()
	if !server.ReadyForConnections(readyWait) {
		server.Shutdown()
		return nil, errNotReady
	}
	return &Broker{server: server}, nil
}

func (b *Broker) ClientURL() string {
	return b.server.ClientURL()
}

func (b *Broker) Shutdown() {
	b.server.Shutdown()
	b.server.WaitForShutdown()
}
