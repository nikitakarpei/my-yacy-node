// Package loopback opens the listeners of the peer that only the local host can reach, and gives
// the URL at which the local host reaches a listener of the peer.
package loopback

import (
	"context"
	"net"
	"net/url"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	Host           = "127.0.0.1"
	AnyFreeAddress = Host + ":0"
)

func Listen(ctx context.Context) (net.Listener, error) {
	return servergroup.Listen(ctx, AnyFreeAddress)
}

func URLOf(listener net.Listener) string {
	address := listener.Addr().(*net.TCPAddr)
	host := address.IP.String()
	if address.IP.IsUnspecified() {
		host = Host
	}
	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, strconv.Itoa(address.Port)),
	}).String()
}
