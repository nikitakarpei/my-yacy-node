// Package proxyprotocolingress owns the private HTTP listener that stands
// between the tunnel and the node. Every connection must start with a PROXY
// protocol header. The ingress replaces all forwarding headers of the request
// with the one address that header authenticates, and it sends the request to
// the node origin from one bound local address.
package proxyprotocolingress

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/pires/go-proxyproto"
)

const (
	loopbackListenAddress    = "127.0.0.1:0"
	listenNetwork            = "tcp"
	readHeaderTimeout        = 10 * time.Second
	idleTimeout              = 60 * time.Second
	maximumRequestHeaderByte = 64 * 1024
	headerReadTimeout        = 5 * time.Second
	forwardedForHeaderName   = "X-Forwarded-For"
	forwardedHeaderPrefix    = "x-forwarded-"
)

var strippedHeaderNames = []string{"Forwarded", "X-Real-Ip"}

type Configuration struct {
	Origin        *url.URL
	EgressAddress netip.Addr
}

type Ingress struct {
	listener net.Listener
	server   *http.Server
	failure  chan error
	port     int
}

func Listen(ctx context.Context, configuration Configuration) (*Ingress, error) {
	listenConfiguration := net.ListenConfig{}

	accepted, err := listenConfiguration.Listen(ctx, listenNetwork, loopbackListenAddress)
	if err != nil {
		return nil, fmt.Errorf("listen on the private ingress: %w", err)
	}

	address, isTCPAddress := accepted.Addr().(*net.TCPAddr)
	if !isTCPAddress {
		return nil, fmt.Errorf("listen on the private ingress: %s is not a TCP address",
			accepted.Addr())
	}

	ingress := &Ingress{
		listener: accepted,
		server: &http.Server{
			Handler:           proxyTo(configuration),
			ReadHeaderTimeout: readHeaderTimeout,
			IdleTimeout:       idleTimeout,
			MaxHeaderBytes:    maximumRequestHeaderByte,
		},
		failure: make(chan error, 1),
		port:    address.Port,
	}
	go ingress.serve()

	slog.DebugContext(ctx, "private ingress listening", slog.Int("port", ingress.port))

	return ingress, nil
}

func proxyTo(configuration Configuration) *httputil.ReverseProxy {
	dialer := net.Dialer{
		LocalAddr: &net.TCPAddr{IP: configuration.EgressAddress.AsSlice()},
	}

	return &httputil.ReverseProxy{
		Rewrite:   rewriteTo(configuration.Origin),
		Transport: &http.Transport{DialContext: dialer.DialContext},
	}
}

func rewriteTo(origin *url.URL) func(*httputil.ProxyRequest) {
	return func(request *httputil.ProxyRequest) {
		request.Out.URL.Scheme = origin.Scheme
		request.Out.URL.Host = origin.Host
		request.Out.Host = request.In.Host

		for name := range request.Out.Header {
			if strings.HasPrefix(strings.ToLower(name), forwardedHeaderPrefix) {
				request.Out.Header.Del(name)
			}
		}

		for _, name := range strippedHeaderNames {
			request.Out.Header.Del(name)
		}

		request.Out.Header.Set(forwardedForHeaderName, sourceAddressOf(request.In.RemoteAddr))
	}
}

func sourceAddressOf(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return remoteAddress
	}

	return host
}

func (ingress *Ingress) serve() {
	err := ingress.server.Serve(headeredListener(ingress.listener))
	ingress.failure <- fmt.Errorf("serve the private ingress: %w", err)
}

func headeredListener(accepted net.Listener) net.Listener {
	return &proxyproto.Listener{
		Listener:          accepted,
		ConnPolicy:        headerRequired,
		ReadHeaderTimeout: headerReadTimeout,
	}
}

func headerRequired(proxyproto.ConnPolicyOptions) (proxyproto.Policy, error) {
	return proxyproto.REQUIRE, nil
}

func (ingress *Ingress) Port() int {
	return ingress.port
}

func (ingress *Ingress) Failure() <-chan error {
	return ingress.failure
}

func (ingress *Ingress) Close() error {
	if err := ingress.server.Close(); err != nil {
		return fmt.Errorf("close the private ingress: %w", err)
	}

	return nil
}
