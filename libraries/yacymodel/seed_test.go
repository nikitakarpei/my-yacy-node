package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func addressableSeed(t *testing.T) yacymodel.Seed {
	t.Helper()
	host, err := yacymodel.ParseHost("192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	port, err := yacymodel.ParsePort("8090")
	if err != nil {
		t.Fatal(err)
	}
	return yacymodel.Seed{PrimaryAddress: yacymodel.Some(host), Port: yacymodel.Some(port)}
}

func TestSeedNetworkAddress(t *testing.T) {
	seed := addressableSeed(t)
	address, ok := seed.NetworkAddress()
	if !ok || address.String() != "192.0.2.1:8090" {
		t.Fatalf("NetworkAddress = %q, %v", address, ok)
	}
}

func TestSeedNetworkAddressMissingPort(t *testing.T) {
	host, _ := yacymodel.ParseHost("192.0.2.1")
	seed := yacymodel.Seed{PrimaryAddress: yacymodel.Some(host)}
	if _, ok := seed.NetworkAddress(); ok {
		t.Fatal("NetworkAddress present without port")
	}
}

func TestSeedHTTPEndpoint(t *testing.T) {
	seed := addressableSeed(t)
	endpoint, err := seed.HTTPEndpoint("/yacy/hello.html")
	if err != nil {
		t.Fatal(err)
	}
	if got := endpoint.String(); got != "http://192.0.2.1:8090/yacy/hello.html" {
		t.Fatalf("HTTPEndpoint = %q", got)
	}
}

func TestSeedHTTPEndpointUnreachable(t *testing.T) {
	if _, err := (yacymodel.Seed{}).HTTPEndpoint("/x"); err == nil {
		t.Fatal("HTTPEndpoint on addressless seed did not fail")
	}
}

func TestSeedIsAddressable(t *testing.T) {
	if (yacymodel.Seed{}).IsAddressable() {
		t.Fatal("empty seed reported addressable")
	}
	if !addressableSeed(t).IsAddressable() {
		t.Fatal("seed with primary address not addressable")
	}
	host, _ := yacymodel.ParseHost("2001:db8::1")
	viaAdditional := yacymodel.Seed{AdditionalAddresses: yacymodel.Some([]yacymodel.Host{host})}
	if !viaAdditional.IsAddressable() {
		t.Fatal("seed with additional address not addressable")
	}
}
