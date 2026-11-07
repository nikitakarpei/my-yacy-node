// Package processnetwork carries the connections between the parts of the peer inside its
// process, so that they listen on no port. A part takes a host name in the network, and the
// other parts dial that name. The network dials any other address on the network of the host.
// Taking a host name twice is a wiring fault and panics.
package processnetwork

import (
	"context"
	"fmt"
	"net"
	"sync"
)

const networkName = "process"

type Network struct {
	mu          sync.Mutex
	connections map[string]func(context.Context) (net.Conn, error)
	hostDialer  net.Dialer
}

func New() *Network {
	return &Network{connections: map[string]func(context.Context) (net.Conn, error){}}
}

func (n *Network) ListenerAt(host string) net.Listener {
	listener := &listener{
		host:        host,
		connections: make(chan net.Conn),
		closed:      make(chan struct{}),
	}
	n.take(host, listener.connectionFor)
	return listener
}

func (n *Network) Attach(host string, connection func() (net.Conn, error)) {
	n.take(host, func(context.Context) (net.Conn, error) { return connection() })
}

func (n *Network) take(host string, connectionFor func(context.Context) (net.Conn, error)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, taken := n.connections[host]; taken {
		panic("host taken twice in the process network: " + host)
	}
	n.connections[host] = connectionFor
}

func (n *Network) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	connectionFor, found := n.connectionsTo(hostOf(address))
	if !found {
		return n.hostConnectionTo(ctx, network, address)
	}
	return connectionFor(ctx)
}

func (n *Network) connectionsTo(host string) (func(context.Context) (net.Conn, error), bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	connectionFor, found := n.connections[host]
	return connectionFor, found
}

func hostOf(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return host
}

func (n *Network) hostConnectionTo(ctx context.Context, network, address string) (net.Conn, error) {
	connection, err := n.hostDialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("dial %s on the host network: %w", address, err)
	}
	return connection, nil
}

type listener struct {
	host        string
	connections chan net.Conn
	closed      chan struct{}
	closeOnce   sync.Once
}

func (l *listener) connectionFor(ctx context.Context) (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case l.connections <- server:
		return client, nil
	case <-l.closed:
		_, _ = client.Close(), server.Close()
		return nil, fmt.Errorf("dial %s: %w", l.host, net.ErrClosed)
	case <-ctx.Done():
		_, _ = client.Close(), server.Close()
		return nil, fmt.Errorf("dial %s: %w", l.host, ctx.Err())
	}
}

func (l *listener) Accept() (net.Conn, error) {
	select {
	case connection := <-l.connections:
		return connection, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *listener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

func (l *listener) Addr() net.Addr {
	return address(l.host)
}

type address string

func (address) Network() string {
	return networkName
}

func (a address) String() string {
	return string(a)
}
