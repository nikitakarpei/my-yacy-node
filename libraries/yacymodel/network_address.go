package yacymodel

import (
	"errors"
	"fmt"
	"net"
)

var ErrBadNetworkAddress = errors.New("bad network address")

type NetworkAddress struct {
	host Host
	port Port
}

func NetworkAddressOf(host Host, port Port) (NetworkAddress, error) {
	if host.String() == "" {
		return NetworkAddress{}, fmt.Errorf("%w: missing host", ErrBadNetworkAddress)
	}
	if port < portMin || port > portMax {
		return NetworkAddress{}, fmt.Errorf("%w: port %d out of range", ErrBadNetworkAddress, port)
	}

	return NetworkAddress{host: host, port: port}, nil
}

func (a NetworkAddress) Host() Host { return a.host }

func (a NetworkAddress) Port() Port { return a.port }

func (a NetworkAddress) String() string {
	return net.JoinHostPort(a.host.String(), a.port.String())
}
