package httpguard

import (
	"context"
	"net"
	"net/http"
	"strings"
)

const forwardedForHeader = "X-Forwarded-For"

type remoteAddrKey struct{}

func WithRemoteAddr(ctx context.Context, addr string) context.Context {
	return context.WithValue(ctx, remoteAddrKey{}, addr)
}

func RemoteAddr(ctx context.Context) string {
	addr, _ := ctx.Value(remoteAddrKey{}).(string)

	return addr
}

type ClientAddressResolver struct {
	trustedProxies []*net.IPNet
}

func NewClientAddressResolver(trustedProxies []*net.IPNet) ClientAddressResolver {
	return ClientAddressResolver{trustedProxies: trustedProxies}
}

func (c ClientAddressResolver) Resolve(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	return c.nearestUntrustedHopFrom(host, forwardedHopsOf(r))
}

func forwardedHopsOf(r *http.Request) []string {
	var hops []string
	for _, line := range r.Header.Values(forwardedForHeader) {
		for hop := range strings.SplitSeq(line, ",") {
			hops = append(hops, strings.TrimSpace(hop))
		}
	}

	return hops
}

func (c ClientAddressResolver) nearestUntrustedHopFrom(host string, forwardedHops []string) string {
	hop := host
	for i := len(forwardedHops) - 1; i >= 0; i-- {
		hopIP := net.ParseIP(hop)
		if hopIP == nil || !ipInAny(hopIP, c.trustedProxies) {
			return hop
		}
		if net.ParseIP(forwardedHops[i]) == nil {
			return hop
		}
		hop = forwardedHops[i]
	}

	return hop
}

func ipInAny(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}
