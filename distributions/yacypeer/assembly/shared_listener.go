package yacypeerassembly

import (
	"net"
	"sync"
)

type sharedListener struct {
	net.Listener
	close func() error
}

func sharedListenerFrom(listener net.Listener) sharedListener {
	return sharedListener{Listener: listener, close: sync.OnceValue(listener.Close)}
}

func (l sharedListener) Close() error {
	return l.close()
}
