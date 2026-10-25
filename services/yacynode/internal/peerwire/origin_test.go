package peerwire_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

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

func originTexts(origins []peerwire.Origin) []string {
	texts := make([]string, 0, len(origins))
	for _, origin := range origins {
		texts = append(texts, origin.String())
	}

	return texts
}

func TestOriginsFromTryEveryHostInOrderAndADomainNameOverHTTPSFirst(t *testing.T) {
	addresses := addressesOf(t, "example.com", "192.0.2.1", "198.51.100.1")
	addresses.SecurePort = yacymodel.Some(yacymodel.Port(8443))

	want := []string{
		"https://example.com:443", "https://example.com:8443", "http://example.com:8090",
		"http://192.0.2.1:8090", "http://198.51.100.1:8090",
	}
	if got := originTexts(peerwire.OriginsFrom(addresses)); !slices.Equal(got, want) {
		t.Fatalf("origins = %v, want %v", got, want)
	}
}

func TestOriginsFromTryTheSecurePortOnceWhenItIs443(t *testing.T) {
	addresses := addressesOf(t, "example.com")
	addresses.SecurePort = yacymodel.Some(yacymodel.Port(443))

	want := []string{"https://example.com:443", "http://example.com:8090"}
	if got := originTexts(peerwire.OriginsFrom(addresses)); !slices.Equal(got, want) {
		t.Fatalf("origins = %v, want %v", got, want)
	}
}

func TestOriginsFromTryAtMostFourHosts(t *testing.T) {
	addresses := addressesOf(t, "192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.4", "192.0.2.5")

	if got := peerwire.OriginsFrom(addresses); len(got) != 4 {
		t.Fatalf("origins = %v, want four", originTexts(got))
	}
}

func TestOriginsFromTryAtMostFiveOrigins(t *testing.T) {
	addresses := addressesOf(t, "a.example", "b.example", "c.example")

	if got := peerwire.OriginsFrom(addresses); len(got) != 5 {
		t.Fatalf("origins = %v, want five", originTexts(got))
	}
}

func TestParseSchemeKnowsHTTPAndHTTPSOnly(t *testing.T) {
	for _, text := range []string{"http", "https"} {
		if scheme, err := peerwire.ParseScheme(text); err != nil || string(scheme) != text {
			t.Errorf("ParseScheme(%q) = %q, %v", text, scheme, err)
		}
	}
	if _, err := peerwire.ParseScheme("ftp"); err == nil {
		t.Error("ParseScheme(ftp) accepted an unknown scheme")
	}
}
