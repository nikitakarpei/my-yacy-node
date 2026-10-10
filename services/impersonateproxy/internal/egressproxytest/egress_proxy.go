// Package egressproxytest runs a stand-in for the egress proxy in tests. It
// opens a CONNECT tunnel to each target, or refuses every tunnel.
package egressproxytest

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

type EgressProxy struct {
	server        *httptest.Server
	lock          sync.Mutex
	tunnelTargets []string
	tunnels       []net.Conn
}

func Start(t testing.TB) *EgressProxy {
	t.Helper()
	proxy := &EgressProxy{}
	proxy.server = httptest.NewServer(http.HandlerFunc(proxy.openTunnel))
	t.Cleanup(proxy.close)
	return proxy
}

func StartRefusing(t testing.TB) *EgressProxy {
	t.Helper()
	proxy := &EgressProxy{}
	proxy.server = httptest.NewServer(http.HandlerFunc(proxy.refuseTunnel))
	t.Cleanup(proxy.close)
	return proxy
}

func (p *EgressProxy) URL() *url.URL {
	address, _ := url.Parse(p.server.URL)
	return address
}

func (p *EgressProxy) TunnelTargets() []string {
	p.lock.Lock()
	defer p.lock.Unlock()
	return append([]string(nil), p.tunnelTargets...)
}

func (p *EgressProxy) openTunnel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var dialer net.Dialer
	target, err := dialer.DialContext(r.Context(), "tcp", r.Host)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	client, _, err := http.NewResponseController(w).Hijack()
	if err != nil {
		_ = target.Close()
		return
	}
	p.record(r.Host, client, target)
	_, _ = io.WriteString(client, "HTTP/1.1 200 Connection established\r\n\r\n")
	go relay(target, client)
	go relay(client, target)
}

func (p *EgressProxy) record(tunnelTarget string, tunnels ...net.Conn) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.tunnelTargets = append(p.tunnelTargets, tunnelTarget)
	p.tunnels = append(p.tunnels, tunnels...)
}

func relay(destination, source net.Conn) {
	_, _ = io.Copy(destination, source)
	_ = destination.Close()
	_ = source.Close()
}

func (p *EgressProxy) refuseTunnel(w http.ResponseWriter, r *http.Request) {
	p.record(r.Host)
	w.WriteHeader(http.StatusForbidden)
}

func (p *EgressProxy) close() {
	p.server.Close()
	p.lock.Lock()
	defer p.lock.Unlock()
	for _, tunnel := range p.tunnels {
		_ = tunnel.Close()
	}
}
