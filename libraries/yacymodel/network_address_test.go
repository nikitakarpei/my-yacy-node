package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestNetworkAddressJoinsHostAndPort(t *testing.T) {
	host, err := yacymodel.ParseHost("2001:db8::1")
	if err != nil {
		t.Fatalf("ParseHost: %v", err)
	}
	address, err := yacymodel.NetworkAddressOf(host, yacymodel.Port(8090))
	if err != nil {
		t.Fatalf("NetworkAddressOf: %v", err)
	}

	if got := address.String(); got != "[2001:db8::1]:8090" {
		t.Errorf("String = %q, want %q", got, "[2001:db8::1]:8090")
	}
	if address.Host() != host {
		t.Errorf("Host = %v, want %v", address.Host(), host)
	}
	if address.Port() != yacymodel.Port(8090) {
		t.Errorf("Port = %v, want 8090", address.Port())
	}
}

func TestNetworkAddressRejectsMissingHost(t *testing.T) {
	if _, err := yacymodel.NetworkAddressOf(yacymodel.Host{}, yacymodel.Port(8090)); err == nil {
		t.Fatal("NetworkAddressOf accepted a missing host")
	}
}

func TestNetworkAddressRejectsInvalidPort(t *testing.T) {
	host, err := yacymodel.ParseHost("peer.example")
	if err != nil {
		t.Fatalf("ParseHost: %v", err)
	}
	if _, err := yacymodel.NetworkAddressOf(host, 0); err == nil {
		t.Fatal("NetworkAddressOf accepted port zero")
	}
}
