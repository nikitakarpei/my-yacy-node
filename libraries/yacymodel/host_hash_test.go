package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
)

func TestHostHashEncodingIsFrozen(t *testing.T) {
	cases := map[string]string{
		"http://example.com/path": "QpK89Y",
		"ftp://ftp.example.com/":  "bareG4",
		"http://localhost/":       "sIW_pc",
		"http://bücher.de/":       "fMfOoC",
	}
	for address, want := range cases {
		if got := urlhashtest.URLHashOf(t, address).HostHash(); got.String() != want {
			t.Errorf("host hash of %q = %q, want %q", address, got.String(), want)
		}
	}
}

func TestHostHashIgnoresThePathTheDefaultPortAndTheHostCase(t *testing.T) {
	plain := urlhashtest.URLHashOf(t, "http://example.com/one").HostHash()
	for _, address := range []string{
		"http://example.com/two",
		"http://example.com:80/two",
		"http://example.com/",
		"http://Example.COM/two",
		"http://EXAMPLE.com:80/",
	} {
		if got := urlhashtest.URLHashOf(t, address).HostHash(); got != plain {
			t.Errorf("host hash of %q = %q, want %q", address, got.String(), plain.String())
		}
	}
}

func TestHostHashDistinguishesTheProtocol(t *testing.T) {
	overHTTP := urlhashtest.URLHashOf(t, "http://ftp.example.com/").HostHash()
	overFTP := urlhashtest.URLHashOf(t, "ftp://ftp.example.com/").HostHash()
	if overHTTP == overFTP {
		t.Errorf("one host on two protocols must not share a host hash, both %q", overHTTP)
	}
}
