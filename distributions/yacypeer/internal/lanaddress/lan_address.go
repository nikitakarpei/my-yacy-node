// Package lanaddress names how a device on the local network reaches a listener of the peer:
// by the address of the network interface that the peer uses for outgoing traffic, and by the
// mDNS name of the host.
package lanaddress

import (
	"context"
	"net"
	"os"
	"strconv"
)

const (
	mdnsSuffix = ".local"
	probeAddr  = "192.0.2.1:9"
)

func Of(listener *net.TCPAddr) string {
	host := listener.IP
	if host.IsUnspecified() {
		host = outgoingIP()
	}
	return net.JoinHostPort(host.String(), strconv.Itoa(listener.Port))
}

func outgoingIP() net.IP {
	probe, err := (&net.Dialer{}).DialContext(context.Background(), "udp", probeAddr)
	if err != nil {
		return net.IPv4(127, 0, 0, 1)
	}
	defer func() { _ = probe.Close() }()
	return probe.LocalAddr().(*net.UDPAddr).IP
}

func Name() string {
	host, err := os.Hostname()
	if err != nil {
		return ""
	}
	return host + mdnsSuffix
}
