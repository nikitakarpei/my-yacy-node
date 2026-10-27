package tcp_test

import (
	"net"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/connecttimers/tcp"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
)

func loopbackListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := loopback.Listen(t.Context())
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func TestConnectTimeOfAListeningAddressIsKnown(t *testing.T) {
	address := loopbackListener(t).Addr().String()

	if _, connected := tcp.New().ConnectTimeOf(t.Context(), address); !connected {
		t.Errorf("%s not connected, want a connect time", address)
	}
}

func TestConnectTimeOfAClosedAddressIsUnknown(t *testing.T) {
	listener := loopbackListener(t)
	address := listener.Addr().String()
	_ = listener.Close()

	if connectTime, connected := tcp.New().ConnectTimeOf(t.Context(), address); connected {
		t.Errorf("%s connected in %s, want no connect time", address, connectTime)
	}
}
