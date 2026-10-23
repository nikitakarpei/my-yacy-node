// Package jetstreamserver runs a NATS server with JetStream inside the service
// process, so a service that owns its streams runs without an external broker.
// Other services reach the server on its address. The server keeps its streams
// in a store directory, so they stay after a restart, and reports what it logs
// through slog.
package jetstreamserver

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
)

const (
	maxPayload     = 8 << 20
	readinessLimit = 10 * time.Second
)

type Config struct {
	Addr     string
	StoreDir string
}

type Server struct {
	server    *natsserver.Server
	clientURL string
}

func Start(config Config) (*Server, error) {
	options, err := optionsFor(config)
	if err != nil {
		return nil, err
	}
	server, err := natsserver.NewServer(options)
	if err != nil {
		return nil, fmt.Errorf("configure the nats server on %s: %w", config.Addr, err)
	}
	server.SetLogger(serverLog{}, false, false)
	go server.Start()
	if !server.ReadyForConnections(readinessLimit) {
		server.Shutdown()
		server.WaitForShutdown()
		return nil, fmt.Errorf(
			"nats server on %s not ready for connections after %s", config.Addr, readinessLimit,
		)
	}
	return &Server{server: server, clientURL: clientURLFor(options.Host, server.Addr())}, nil
}

func optionsFor(config Config) (*natsserver.Options, error) {
	host, portText, err := net.SplitHostPort(config.Addr)
	if err != nil {
		return nil, fmt.Errorf("read the nats server address %q: %w", config.Addr, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nil, fmt.Errorf("read the nats server port %q: %w", portText, err)
	}
	if port == 0 {
		port = natsserver.RANDOM_PORT
	}
	return &natsserver.Options{
		Host:       host,
		Port:       port,
		JetStream:  true,
		StoreDir:   config.StoreDir,
		MaxPayload: maxPayload,
		NoSigs:     true,
	}, nil
}

func clientURLFor(host string, listener net.Addr) string {
	if host == "" || net.ParseIP(host).IsUnspecified() {
		host = net.IPv4(127, 0, 0, 1).String()
	}
	_, port, _ := net.SplitHostPort(listener.String())
	clientURL := url.URL{Scheme: "nats", Host: net.JoinHostPort(host, port)}
	return clientURL.String()
}

func (s *Server) ClientURL() string {
	return s.clientURL
}

func (s *Server) Shutdown() {
	s.server.Shutdown()
	s.server.WaitForShutdown()
}
