package http

import "fmt"

type ProxyDialMode int

const (
	ProxyDialTunnel ProxyDialMode = iota
	ProxyDialAbsoluteURL
)

var proxyDialModeByName = map[string]ProxyDialMode{
	"tunnel":       ProxyDialTunnel,
	"absolute-url": ProxyDialAbsoluteURL,
}

func ProxyDialModeNamed(name string) (ProxyDialMode, error) {
	mode, named := proxyDialModeByName[name]
	if !named {
		return 0, fmt.Errorf("unknown proxy dial mode %q", name)
	}
	return mode, nil
}
