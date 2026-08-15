package yacymodel_test

import "testing"

func TestHostHashEncodingIsFrozen(t *testing.T) {
	cases := map[string]string{
		"http://example.com/path": "QpK89Y",
		"ftp://ftp.example.com/":  "bareG4",
		"http://localhost/":       "sIW_pc",
		"http://bücher.de/":       "fMfOoC",
	}
	for address, want := range cases {
		if got := normalformOfAddress(t, address).HostHash(); got.String() != want {
			t.Errorf("host hash of %q = %q, want %q", address, got.String(), want)
		}
	}
}

func TestHostHashIgnoresThePathTheDefaultPortAndTheHostCase(t *testing.T) {
	plain := normalformOfAddress(t, "http://example.com/one").HostHash()
	for _, address := range []string{
		"http://example.com/two",
		"http://example.com:80/two",
		"http://example.com/",
		"http://Example.COM/two",
		"http://EXAMPLE.com:80/",
	} {
		if got := normalformOfAddress(t, address).HostHash(); got != plain {
			t.Errorf("host hash of %q = %q, want %q", address, got.String(), plain.String())
		}
	}
}

func TestHostHashDistinguishesTheProtocol(t *testing.T) {
	overHTTP := normalformOfAddress(t, "http://ftp.example.com/").HostHash()
	overFTP := normalformOfAddress(t, "ftp://ftp.example.com/").HostHash()
	if overHTTP == overFTP {
		t.Errorf("one host on two protocols must not share a host hash, both %q", overHTTP)
	}
}
