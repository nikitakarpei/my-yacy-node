package yacymodel_test

import (
	"net/netip"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestIsLocalNetworkAddressHoldsForPrivateLoopbackLinkLocalAndSharedAddresses(t *testing.T) {
	for address, wantLocal := range map[string]bool{
		"127.0.0.1":     true,
		"10.25.0.50":    true,
		"172.24.0.1":    true,
		"192.168.3.2":   true,
		"100.71.17.183": true,
		"169.254.1.1":   true,
		"fd00::1":       true,
		"fe80::1":       true,
		"203.0.113.9":   false,
		"2001:db8::1":   false,
	} {
		if got := yacymodel.IsLocalNetworkAddress(netip.MustParseAddr(address)); got != wantLocal {
			t.Errorf("IsLocalNetworkAddress(%s) = %t, want %t", address, got, wantLocal)
		}
	}
}
