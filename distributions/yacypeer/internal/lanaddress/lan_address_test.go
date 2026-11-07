package lanaddress_test

import (
	"net"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/lanaddress"
)

func TestAListenerOnOneAddressIsReachedThere(t *testing.T) {
	address := lanaddress.Of(&net.TCPAddr{IP: net.IPv4(192, 168, 1, 20), Port: 4223})

	if address != "192.168.1.20:4223" {
		t.Errorf("address = %q, want 192.168.1.20:4223", address)
	}
}

func TestAListenerOnEveryAddressIsReachedOnAConcreteOne(t *testing.T) {
	address := lanaddress.Of(&net.TCPAddr{IP: net.IPv4zero, Port: 4223})

	host, port, err := net.SplitHostPort(address)
	if err != nil || port != "4223" || net.ParseIP(host).IsUnspecified() {
		t.Errorf("address = %q, want a concrete host on port 4223", address)
	}
}

func TestTheNameIsAnMDNSName(t *testing.T) {
	if name := lanaddress.Name(); !strings.HasSuffix(name, ".local") {
		t.Errorf("name = %q, want an mDNS name", name)
	}
}
