package asmap_test

import (
	"net/netip"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/asmap"
)

func TestBitcoinCoreMapsAddressesToTheirAutonomousSystem(t *testing.T) {
	for address, wantASN := range map[string]asmap.ASN{
		"1.1.1.1":              13335,
		"8.8.8.8":              15169,
		"9.9.9.9":              19281,
		"::ffff:8.8.4.4":       15169,
		"2606:4700:4700::1111": 13335,
		"2001:4860:4860::8888": 15169,
	} {
		if got := asmap.BitcoinCore().ASNOf(netip.MustParseAddr(address)); got != wantASN {
			t.Errorf("ASNOf(%s) = %d, want %d", address, got, wantASN)
		}
	}
}

func TestBitcoinCoreMapsUnroutableAddressesToASNZero(t *testing.T) {
	for _, address := range []string{
		"10.0.0.1", "::ffff:192.168.1.1", "127.0.0.1", "fe80::1", "fd00::1",
	} {
		if got := asmap.BitcoinCore().ASNOf(netip.MustParseAddr(address)); got != 0 {
			t.Errorf("ASNOf(%s) = %d, want 0", address, got)
		}
	}
}
