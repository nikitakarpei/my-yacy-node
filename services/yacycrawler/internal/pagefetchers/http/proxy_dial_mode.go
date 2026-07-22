package http

type ProxyDialMode int

const (
	ProxyDialTunnel ProxyDialMode = iota
	ProxyDialAbsoluteURL
)
