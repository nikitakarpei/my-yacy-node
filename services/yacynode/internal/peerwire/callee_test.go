package peerwire_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

func TestIdentifiedCalleeHasTheHashOfItsIdentity(t *testing.T) {
	callee := peerwire.IdentifiedCallee(peerkeytest.Key(0).Identity())

	if callee.Hash() != peerkeytest.Key(0).Hash() {
		t.Errorf("Hash = %v, want %v", callee.Hash(), peerkeytest.Key(0).Hash())
	}
	if identity, _ := callee.Identity().Get(); identity != peerkeytest.Key(0).Identity() {
		t.Errorf("Identity = %v, want %v", identity, peerkeytest.Key(0).Identity())
	}
}

func TestUnidentifiedCalleeHasOnlyItsHash(t *testing.T) {
	callee := peerwire.UnidentifiedCallee(peerkeytest.Key(1).Hash())

	if callee.Hash() != peerkeytest.Key(1).Hash() ||
		callee.Identity() != yacymodel.None[yacymodel.PeerIdentity]() {
		t.Errorf("callee = %v, want hash %v without identity", callee, peerkeytest.Key(1).Hash())
	}
}

var unidentified = peerwire.UnidentifiedCallee(peerkeytest.Key(0).Hash())

var identified = peerwire.IdentifiedCallee(peerkeytest.Key(0).Identity())

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
	if got := baseURLTexts(
		unidentified.BaseURLsFrom(addresses),
	); !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("base URLs = %v, want %v", got, want)
	}
}

func TestBaseURLsFromTryTheSecurePortOnceWhenItIs443(t *testing.T) {
	addresses := addressesOf(t, "example.com")
	addresses.SecurePort = yacymodel.Some(yacymodel.Port(443))

	want := []string{"https://example.com:443", "http://example.com:8090"}
	if got := baseURLTexts(
		unidentified.BaseURLsFrom(addresses),
	); !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("base URLs = %v, want %v", got, want)
	}
}

func TestBaseURLsFromTryAtMostFourHosts(t *testing.T) {
	addresses := addressesOf(t, "192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.4", "192.0.2.5")

	if got := unidentified.BaseURLsFrom(addresses); len(got) != 4 {
		t.Fatalf("base URLs = %v, want four", baseURLTexts(got))
	}
}

func TestBaseURLsFromTryAtMostFiveBaseURLs(t *testing.T) {
	addresses := addressesOf(t, "a.example", "b.example", "c.example")

	if got := unidentified.BaseURLsFrom(addresses); len(got) != 5 {
		t.Fatalf("base URLs = %v, want five", baseURLTexts(got))
	}
}

func TestBaseURLsOfAnIdentifiedPeerTryOnlyHTTPSAtTheSecurePortOfEveryHost(t *testing.T) {
	addresses := addressesOf(
		t,
		"example.com",
		"192.0.2.1",
		"198.51.100.1",
		"203.0.113.1",
		"203.0.113.2",
	)
	addresses.SecurePort = yacymodel.Some(yacymodel.Port(8443))

	want := []string{
		"https://example.com:8443", "https://192.0.2.1:8443",
		"https://198.51.100.1:8443", "https://203.0.113.1:8443",
	}
	if got := baseURLTexts(identified.BaseURLsFrom(addresses)); !slices.Equal(got, want) {
		t.Fatalf("base URLs = %v, want %v", got, want)
	}
}

func TestBaseURLsOfAnIdentifiedPeerWithoutASecurePortTryOnlyHTTP(t *testing.T) {
	addresses := addressesOf(t, "example.com", "192.0.2.1")

	want := []string{"http://example.com:8090", "http://192.0.2.1:8090"}
	if got := baseURLTexts(identified.BaseURLsFrom(addresses)); !slices.Equal(got, want) {
		t.Fatalf("base URLs = %v, want %v", got, want)
	}
}
