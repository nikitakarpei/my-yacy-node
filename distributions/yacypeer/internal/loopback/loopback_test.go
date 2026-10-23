package loopback_test

import (
	"net"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
)

func TestListenBindsAFreePortOfTheLoopbackHost(t *testing.T) {
	listener, err := loopback.Listen(t.Context())
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split %q: %v", listener.Addr(), err)
	}
	if host != loopback.Host || port == "0" {
		t.Errorf("address = %s, want a free port of %s", listener.Addr(), loopback.Host)
	}
}
