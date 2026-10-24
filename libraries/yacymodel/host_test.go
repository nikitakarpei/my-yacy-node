package yacymodel_test

import (
	"net/netip"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestHostFromAnAddressKeepsTheAddress(t *testing.T) {
	addr := netip.MustParseAddr("203.0.113.9")
	host := yacymodel.HostFrom(addr)

	if got, isAddr := host.Addr(); !isAddr || got != addr {
		t.Errorf("Addr = %v (address %t), want %v", got, isAddr, addr)
	}
	if host.String() != "203.0.113.9" {
		t.Errorf("String = %q, want 203.0.113.9", host.String())
	}
}

func TestAHostnameHasNoAddress(t *testing.T) {
	host, err := yacymodel.ParseHost("peer.example")
	if err != nil {
		t.Fatalf("ParseHost: %v", err)
	}

	if addr, isAddr := host.Addr(); isAddr {
		t.Errorf("Addr = %v, want none for a hostname", addr)
	}
}
