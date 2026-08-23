package http_test

import (
	"testing"

	httppkg "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
)

func TestProxyDialModeNamed(t *testing.T) {
	named := map[string]httppkg.ProxyDialMode{
		"tunnel":       httppkg.ProxyDialTunnel,
		"absolute-url": httppkg.ProxyDialAbsoluteURL,
	}
	for name, want := range named {
		mode, err := httppkg.ProxyDialModeNamed(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if mode != want {
			t.Errorf("%s = %d, want %d", name, mode, want)
		}
	}
}

func TestProxyDialModeNamedRejectsAnUnknownName(t *testing.T) {
	if _, err := httppkg.ProxyDialModeNamed("carrier-pigeon"); err == nil {
		t.Fatal("an unknown proxy dial mode should be rejected")
	}
}
