package thishost_test

import (
	"net"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/thishost"
)

func TestThisHostListsAtLeastOneAddress(t *testing.T) {
	everyAddress := &net.TCPAddr{IP: net.IPv4zero, Port: 4223}

	if addresses := thishost.Host().AddressesFor(everyAddress); len(addresses) == 0 {
		t.Error("this host lists no address")
	}
}
