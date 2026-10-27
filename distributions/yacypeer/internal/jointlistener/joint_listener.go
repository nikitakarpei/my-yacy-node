// Package jointlistener accepts the connections of several listeners as one listener, so one
// server serves its endpoint on every network that the peer joins.
package jointlistener

import (
	"errors"
	"net"
	"sync"
)

type Listener struct {
	listeners   []net.Listener
	acceptances chan acceptance
	closed      chan struct{}
	closeOnce   sync.Once
}

type acceptance struct {
	conn net.Conn
	err  error
}

func New(listeners ...net.Listener) *Listener {
	joint := &Listener{
		listeners:   listeners,
		acceptances: make(chan acceptance),
		closed:      make(chan struct{}),
	}
	for _, listener := range listeners {
		go joint.acceptFrom(listener)
	}
	return joint
}

func (l *Listener) acceptFrom(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		select {
		case l.acceptances <- acceptance{conn: conn, err: err}:
		case <-l.closed:
			if conn != nil {
				_ = conn.Close()
			}
			return
		}
		if err != nil {
			return
		}
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case accepted := <-l.acceptances:
		return accepted.conn, accepted.err
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *Listener) Close() error {
	var failures []error
	l.closeOnce.Do(func() {
		close(l.closed)
		for _, listener := range l.listeners {
			failures = append(failures, listener.Close())
		}
	})
	return errors.Join(failures...)
}

func (l *Listener) Addr() net.Addr {
	return l.listeners[0].Addr()
}
