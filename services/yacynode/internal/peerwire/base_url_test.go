package peerwire_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

var unsigned = yacymodel.None[yacymodel.SignedIdentity]()

var signed = yacymodel.Some(yacymodel.SignedIdentity{Identity: peerkeytest.Key(0).Identity()})

func addressesOf(t *testing.T, hosts ...string) yacymodel.PeerAddresses {
	t.Helper()

	addresses := yacymodel.PeerAddresses{Port: 8090}
	for _, text := range hosts {
		host, err := yacymodel.ParseHost(text)
		if err != nil {
			t.Fatalf("parse host: %v", err)
		}
		addresses.Hosts = append(addresses.Hosts, host)
	}

	return addresses
}

func baseURLTexts(baseURLs []peerwire.BaseURL) []string {
	texts := make([]string, 0, len(baseURLs))
	for _, baseURL := range baseURLs {
		texts = append(texts, baseURL.String())
	}

	return texts
}

func TestBaseURLsFromTryEveryHostInOrderAndADomainNameOverHTTPSFirst(t *testing.T) {
	addresses := addressesOf(t, "example.com", "192.0.2.1", "198.51.100.1")
	addresses.SecurePort = yacymodel.Some(yacymodel.Port(8443))

	want := []string{
		"https://example.com:443", "https://example.com:8443", "http://example.com:8090",
		"http://192.0.2.1:8090", "http://198.51.100.1:8090",
	}
	if got := baseURLTexts(peerwire.BaseURLsFrom(addresses, unsigned)); !slices.Equal(got, want) {
		t.Fatalf("base URLs = %v, want %v", got, want)
	}
}

func TestBaseURLsFromTryTheSecurePortOnceWhenItIs443(t *testing.T) {
	addresses := addressesOf(t, "example.com")
	addresses.SecurePort = yacymodel.Some(yacymodel.Port(443))

	want := []string{"https://example.com:443", "http://example.com:8090"}
	if got := baseURLTexts(peerwire.BaseURLsFrom(addresses, unsigned)); !slices.Equal(got, want) {
		t.Fatalf("base URLs = %v, want %v", got, want)
	}
}

func TestBaseURLsFromTryAtMostFourHosts(t *testing.T) {
	addresses := addressesOf(t, "192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.4", "192.0.2.5")

	if got := peerwire.BaseURLsFrom(addresses, unsigned); len(got) != 4 {
		t.Fatalf("base URLs = %v, want four", baseURLTexts(got))
	}
}

func TestBaseURLsFromTryAtMostFiveBaseURLs(t *testing.T) {
	addresses := addressesOf(t, "a.example", "b.example", "c.example")

	if got := peerwire.BaseURLsFrom(addresses, unsigned); len(got) != 5 {
		t.Fatalf("base URLs = %v, want five", baseURLTexts(got))
	}
}

func TestBaseURLsFromASignedSeedTryOnlyPeerHTTPSAtTheSecurePortOfEveryHost(t *testing.T) {
	addresses := addressesOf(
		t,
		"example.com",
		"192.0.2.1",
		"198.51.100.1",
		"203.0.113.1",
		"203.0.113.2",
	)
	addresses.SecurePort = yacymodel.Some(yacymodel.Port(8443))

	baseURLs := peerwire.BaseURLsFrom(addresses, signed)

	want := []string{
		"https://example.com:8443", "https://192.0.2.1:8443",
		"https://198.51.100.1:8443", "https://203.0.113.1:8443",
	}
	if got := baseURLTexts(baseURLs); !slices.Equal(got, want) {
		t.Fatalf("base URLs = %v, want %v", got, want)
	}
	for _, baseURL := range baseURLs {
		if baseURL.Scheme != peerwire.PeerHTTPS {
			t.Errorf("scheme of %s = %q, want %q", baseURL, baseURL.Scheme, peerwire.PeerHTTPS)
		}
	}
}

func TestBaseURLsFromASignedSeedWithoutASecurePortKeepTheUnsignedRule(t *testing.T) {
	addresses := addressesOf(t, "example.com", "192.0.2.1")

	got := baseURLTexts(peerwire.BaseURLsFrom(addresses, signed))
	if want := baseURLTexts(peerwire.BaseURLsFrom(addresses, unsigned)); !slices.Equal(got, want) {
		t.Fatalf("base URLs = %v, want %v", got, want)
	}
}

func TestParseSchemeKnowsHTTPHTTPSAndPeerHTTPSOnly(t *testing.T) {
	for _, text := range []string{"http", "https", "peer-https"} {
		if scheme, err := peerwire.ParseScheme(text); err != nil || string(scheme) != text {
			t.Errorf("ParseScheme(%q) = %q, %v", text, scheme, err)
		}
	}
	if _, err := peerwire.ParseScheme("ftp"); err == nil {
		t.Error("ParseScheme(ftp) accepted an unknown scheme")
	}
}
