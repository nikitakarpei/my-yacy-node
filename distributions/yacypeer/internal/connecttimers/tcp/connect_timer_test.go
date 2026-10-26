package tcp_test

import (
	"net"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/connecttimers/tcp"
)

func listeningAddress(t *testing.T) string {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener.Addr().String()
}

func closedAddress(t *testing.T) string {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func TestConnectTimeOfAListeningAddressIsKnown(t *testing.T) {
	address := listeningAddress(t)

	if _, connected := tcp.New().ConnectTimeOf(t.Context(), address); !connected {
		t.Errorf("%s not connected, want a connect time", address)
	}
}

func TestConnectTimeOfAClosedAddressIsUnknown(t *testing.T) {
	address := closedAddress(t)

	if connectTime, connected := tcp.New().ConnectTimeOf(t.Context(), address); connected {
		t.Errorf("%s connected in %s, want no connect time", address, connectTime)
	}
}
