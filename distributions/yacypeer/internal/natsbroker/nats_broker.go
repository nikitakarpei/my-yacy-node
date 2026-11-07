// Package natsbroker runs the NATS JetStream server built into the peer. The parts of the peer
// connect to it inside the process. For companions it can also listen on a TCP address with TLS,
// where an authentication that the peer gives decides who connects and what each one may do.
package natsbroker

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
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

type CompanionListening struct {
	Authentication natsserver.Authentication
	Address        string
	Certificate    tls.Certificate
}

func Start(storeDirectory string) (*Broker, error) {
	return started(&natsserver.Options{
		DontListen: true,
		MaxPayload: payloadCeiling,
		JetStream:  true,
		StoreDir:   storeDirectory,
	})
}

func StartForCompanions(storeDirectory string, companions CompanionListening) (*Broker, error) {
	host, port, err := hostAndPortOf(companions.Address)
	if err != nil {
		return nil, err
	}
	return started(&natsserver.Options{
		Host:       host,
		Port:       port,
		MaxPayload: payloadCeiling,
		JetStream:  true,
		StoreDir:   storeDirectory,
		TLS:        true,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{companions.Certificate},
			ClientAuth:   tls.RequestClientCert,
			MinVersion:   tls.VersionTLS13,
		},
		CustomClientAuthentication: companions.Authentication,
	})
}

func hostAndPortOf(address string) (string, int, error) {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, fmt.Errorf("companion address: %w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return "", 0, fmt.Errorf("companion address: %w", err)
	}
	if port == 0 {
		return host, natsserver.RANDOM_PORT, nil
	}
	return host, port, nil
}

func started(options *natsserver.Options) (*Broker, error) {
	server, err := natsserver.NewServer(options)
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

func (b *Broker) CompanionAddress() net.Addr {
	return b.server.Addr()
}

func (b *Broker) ConnectedUsers() map[string]bool {
	connections, err := b.server.Connz(&natsserver.ConnzOptions{Username: true})
	if err != nil {
		return map[string]bool{}
	}
	users := make(map[string]bool, len(connections.Conns))
	for _, connection := range connections.Conns {
		users[connection.AuthorizedUser] = true
	}
	return users
}

func (b *Broker) DisconnectUser(user string) {
	connections, err := b.server.Connz(&natsserver.ConnzOptions{User: user})
	if err != nil {
		return
	}
	for _, connection := range connections.Conns {
		_ = b.server.DisconnectClientByID(connection.Cid)
	}
}

func (b *Broker) Shutdown() {
	b.server.Shutdown()
	b.server.WaitForShutdown()
}
