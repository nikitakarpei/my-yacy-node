// Package peerport serves the peer protocol over HTTP and HTTPS on one port. It
// hands each connection of the port to the HTTPS listener when its first byte
// starts a TLS handshake, and to the HTTP listener otherwise. It closes a
// connection that sends no byte in time, and reports the scheme of each
// connection it hands over. The port closes when both listeners close.
package peerport

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

const (
	tlsHandshakeRecord byte = 0x16
	firstByteTimeout        = 10 * time.Second
)

type Scheme string

const (
	HTTPScheme  Scheme = "http"
	HTTPSScheme Scheme = "https"
)

type Clock interface {
	Now() time.Time
}

type ConnectionObserver interface {
	ObserveConnection(scheme Scheme)
}

type Listeners struct {
	HTTP  net.Listener
	HTTPS net.Listener
}

func ListenersOn(
	ctx context.Context,
	peerPort net.Listener,
	clock Clock,
	observer ConnectionObserver,
) Listeners {
	port := &sharedPort{
		peerPort:      peerPort,
		clock:         clock,
		observer:      observer,
		stopped:       make(chan struct{}),
		openListeners: 2,
	}
	port.httpListener = newSchemeListener(port)
	port.httpsListener = newSchemeListener(port)
	go port.acceptAll(ctx)

	return Listeners{HTTP: port.httpListener, HTTPS: port.httpsListener}
}

type sharedPort struct {
	peerPort      net.Listener
	clock         Clock
	observer      ConnectionObserver
	httpListener  *schemeListener
	httpsListener *schemeListener
	stopped       chan struct{}
	stopReason    error

	mu            sync.Mutex
	openListeners int
}

func (p *sharedPort) acceptAll(ctx context.Context) {
	for {
		connection, err := p.peerPort.Accept()
		if err != nil {
			p.stopReason = fmt.Errorf("accept on the peer port: %w", err)
			close(p.stopped)

			return
		}
		go p.handOver(ctx, connection)
	}
}

func (p *sharedPort) handOver(ctx context.Context, connection net.Conn) {
	firstByte, err := p.firstByteOf(connection)
	if err != nil {
		slog.DebugContext(ctx, "peer connection closed before its first byte",
			slog.String("remoteAddr", connection.RemoteAddr().String()),
			slog.Any("error", err),
		)
		_ = connection.Close()

		return
	}
	scheme, listener := HTTPScheme, p.httpListener
	if firstByte == tlsHandshakeRecord {
		scheme, listener = HTTPSScheme, p.httpsListener
	}
	p.observer.ObserveConnection(scheme)
	select {
	case listener.connections <- &replayingConnection{Conn: connection, unread: []byte{firstByte}}:
	case <-listener.closed:
		_ = connection.Close()
	case <-p.stopped:
		_ = connection.Close()
	}
}

func (p *sharedPort) firstByteOf(connection net.Conn) (byte, error) {
	if err := connection.SetReadDeadline(p.clock.Now().Add(firstByteTimeout)); err != nil {
		return 0, fmt.Errorf("set the first byte deadline: %w", err)
	}
	var firstByte [1]byte
	if _, err := io.ReadFull(connection, firstByte[:]); err != nil {
		return 0, fmt.Errorf("read the first byte: %w", err)
	}
	if err := connection.SetReadDeadline(time.Time{}); err != nil {
		return 0, fmt.Errorf("clear the first byte deadline: %w", err)
	}

	return firstByte[0], nil
}

func (p *sharedPort) release() error {
	p.mu.Lock()
	p.openListeners--
	lastListener := p.openListeners == 0
	p.mu.Unlock()
	if !lastListener {
		return nil
	}
	if err := p.peerPort.Close(); err != nil {
		return fmt.Errorf("close the peer port: %w", err)
	}

	return nil
}

type schemeListener struct {
	port        *sharedPort
	connections chan net.Conn
	closed      chan struct{}
	closeOnce   sync.Once
	closeErr    error
}

func newSchemeListener(port *sharedPort) *schemeListener {
	return &schemeListener{
		port:        port,
		connections: make(chan net.Conn),
		closed:      make(chan struct{}),
	}
}

func (l *schemeListener) Accept() (net.Conn, error) {
	select {
	case connection := <-l.connections:
		return connection, nil
	case <-l.closed:
		return nil, net.ErrClosed
	case <-l.port.stopped:
		return nil, l.port.stopReason
	}
}

func (l *schemeListener) Close() error {
	l.closeOnce.Do(func() {
		close(l.closed)
		l.closeErr = l.port.release()
	})

	return l.closeErr
}

func (l *schemeListener) Addr() net.Addr {
	return l.port.peerPort.Addr()
}

type replayingConnection struct {
	net.Conn
	unread []byte
}

func (c *replayingConnection) Read(buffer []byte) (int, error) {
	if len(c.unread) == 0 {
		//nolint:wrapcheck // io.EOF reaches the caller of a reader unwrapped
		return c.Conn.Read(buffer)
	}
	read := copy(buffer, c.unread)
	c.unread = c.unread[read:]

	return read, nil
}
