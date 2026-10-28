package httpguard_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
)

func resolverTrusting(t *testing.T, cidrs ...string) httpguard.ClientAddressResolver {
	t.Helper()
	networks := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			t.Fatalf("parse %s: %v", cidr, err)
		}
		networks = append(networks, network)
	}
	return httpguard.NewClientAddressResolver(networks)
}

func requestFrom(remoteAddr string, forwardedFor ...string) *http.Request {
	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/yacy/hello.html",
		nil,
	)
	request.RemoteAddr = remoteAddr
	for _, line := range forwardedFor {
		request.Header.Add("X-Forwarded-For", line)
	}
	return request
}

func TestResolveGivesTheNearestHopThatIsNotATrustedProxy(t *testing.T) {
	resolver := resolverTrusting(t, "10.0.0.0/8")

	for name, test := range map[string]struct {
		request *http.Request
		want    string
	}{
		"a peer without a proxy": {
			requestFrom("203.0.113.7:4321"),
			"203.0.113.7",
		},
		"a peer that forges the header without a proxy": {
			requestFrom("203.0.113.7:4321", "198.51.100.6"),
			"203.0.113.7",
		},
		"a peer behind a trusted proxy": {
			requestFrom("10.0.0.2:4321", "203.0.113.7"),
			"203.0.113.7",
		},
		"a peer that forges the header behind a proxy that appends": {
			requestFrom("10.0.0.2:4321", "198.51.100.6, 203.0.113.7"),
			"203.0.113.7",
		},
		"a peer behind two trusted proxies": {
			requestFrom("10.0.0.2:4321", "203.0.113.7, 10.0.0.3"),
			"203.0.113.7",
		},
		"a peer behind a proxy that adds a header line": {
			requestFrom("10.0.0.2:4321", "198.51.100.6", "203.0.113.7"),
			"203.0.113.7",
		},
		"a trusted proxy without the header": {
			requestFrom("10.0.0.2:4321"),
			"10.0.0.2",
		},
		"a trusted proxy that forwards no address": {
			requestFrom("10.0.0.2:4321", "unknown"),
			"10.0.0.2",
		},
		"only trusted proxies": {
			requestFrom("10.0.0.2:4321", "10.0.0.4, 10.0.0.3"),
			"10.0.0.4",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := resolver.Resolve(test.request); got != test.want {
				t.Errorf("Resolve = %q, want %q", got, test.want)
			}
		})
	}
}
