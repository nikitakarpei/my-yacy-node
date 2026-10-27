package jointlistener_test

import (
	"errors"
	"net"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/jointlistener"
)

func loopbackListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return listener
}

func dialed(t *testing.T, listener net.Listener) net.Conn {
	t.Helper()
	conn, err := new(net.Dialer).DialContext(t.Context(), "tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial %s: %v", listener.Addr(), err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestAcceptTakesTheConnectionsOfEveryListener(t *testing.T) {
	first, second := loopbackListener(t), loopbackListener(t)
	joint := jointlistener.New(first, second)
	t.Cleanup(func() { _ = joint.Close() })
	callers := map[string]bool{
		dialed(t, first).LocalAddr().String():  true,
		dialed(t, second).LocalAddr().String(): true,
	}

	for range 2 {
		conn, err := joint.Accept()
		if err != nil {
			t.Fatalf("Accept: %v", err)
		}
		if !callers[conn.RemoteAddr().String()] {
			t.Errorf("accepted a connection from %s, want one of %v", conn.RemoteAddr(), callers)
		}
		_ = conn.Close()
	}
}

func TestAddrNamesTheFirstListener(t *testing.T) {
	first, second := loopbackListener(t), loopbackListener(t)
	joint := jointlistener.New(first, second)
	t.Cleanup(func() { _ = joint.Close() })

	if joint.Addr() != first.Addr() {
		t.Errorf("Addr = %s, want %s", joint.Addr(), first.Addr())
	}
}

func TestCloseReleasesTheAddressOfEveryListener(t *testing.T) {
	first, second := loopbackListener(t), loopbackListener(t)
	joint := jointlistener.New(first, second)

	if err := joint.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	for _, listener := range []net.Listener{first, second} {
		released, err := new(net.ListenConfig).Listen(t.Context(), "tcp", listener.Addr().String())
		if err != nil {
			t.Errorf("listen again on %s: %v", listener.Addr(), err)
			continue
		}
		_ = released.Close()
	}
}

func TestAcceptAfterCloseReportsTheListenerClosed(t *testing.T) {
	joint := jointlistener.New(loopbackListener(t))
	if err := joint.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := joint.Accept(); !errors.Is(err, net.ErrClosed) {
		t.Errorf("Accept = %v, want net.ErrClosed", err)
	}
}

func TestASecondCloseClosesNothingAgain(t *testing.T) {
	joint := jointlistener.New(loopbackListener(t))
	if err := joint.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := joint.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

type failingListener struct {
	net.Listener
	failure error
}

func (l failingListener) Accept() (net.Conn, error) {
	return nil, l.failure
}

func TestAcceptReportsTheFailureOfAListener(t *testing.T) {
	failure := errors.New("accept failed")
	joint := jointlistener.New(failingListener{Listener: loopbackListener(t), failure: failure})
	t.Cleanup(func() { _ = joint.Close() })

	if _, err := joint.Accept(); !errors.Is(err, failure) {
		t.Errorf("Accept = %v, want %v", err, failure)
	}
}
