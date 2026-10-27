package loopback_test

import (
	"net"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
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

func TestURLOfReachesAListenerOfEveryHostThroughTheLoopbackHost(t *testing.T) {
	listener, err := servergroup.Listen(t.Context(), ":0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split %q: %v", listener.Addr(), err)
	}
	if got, want := loopback.URLOf(listener), "http://"+loopback.Host+":"+port; got != want {
		t.Errorf("URLOf = %q, want %q", got, want)
	}
}

func TestURLOfKeepsTheHostOfAListener(t *testing.T) {
	listener, err := loopback.Listen(t.Context())
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	if got, want := loopback.URLOf(listener), "http://"+listener.Addr().String(); got != want {
		t.Errorf("URLOf = %q, want %q", got, want)
	}
}

func TestURLAtReachesAnAddressOfEveryHostThroughTheLoopbackHost(t *testing.T) {
	for _, address := range []string{":8097", "0.0.0.0:8097", "[::]:8097"} {
		if got, want := loopback.URLAt(address), "http://"+loopback.Host+":8097"; got != want {
			t.Errorf("URLAt(%q) = %q, want %q", address, got, want)
		}
	}
}

func TestURLAtKeepsTheHostOfAnAddress(t *testing.T) {
	if got, want := loopback.URLAt("192.0.2.7:8097"), "http://192.0.2.7:8097"; got != want {
		t.Errorf("URLAt = %q, want %q", got, want)
	}
}
