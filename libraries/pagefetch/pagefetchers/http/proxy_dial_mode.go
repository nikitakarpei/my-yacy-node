package http

import "fmt"

type ProxyDialMode int

const (
	ProxyDialTunnel ProxyDialMode = iota
	ProxyDialAbsoluteURL
)

const (
	ProxyDialTunnelName      = "tunnel"
	ProxyDialAbsoluteURLName = "absolute-url"
)

var proxyDialModeByName = map[string]ProxyDialMode{
	ProxyDialTunnelName:      ProxyDialTunnel,
	ProxyDialAbsoluteURLName: ProxyDialAbsoluteURL,
}

func ProxyDialModeNamed(name string) (ProxyDialMode, error) {
	mode, named := proxyDialModeByName[name]
	if !named {
		return 0, fmt.Errorf("unknown proxy dial mode %q", name)
	}
	return mode, nil
}
