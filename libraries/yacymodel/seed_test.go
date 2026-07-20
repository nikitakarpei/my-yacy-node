package yacymodel

import "testing"

func addressableSeed(t *testing.T) Seed {
	t.Helper()
	host, err := ParseHost("192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	port, err := ParsePort("8090")
	if err != nil {
		t.Fatal(err)
	}
	return Seed{PrimaryAddress: Some(host), Port: Some(port)}
}

func TestSeedNetworkAddress(t *testing.T) {
	seed := addressableSeed(t)
	address, ok := seed.NetworkAddress()
	if !ok || address != "192.0.2.1:8090" {
		t.Fatalf("NetworkAddress = %q, %v", address, ok)
	}
}

func TestSeedNetworkAddressMissingPort(t *testing.T) {
	host, _ := ParseHost("192.0.2.1")
	seed := Seed{PrimaryAddress: Some(host)}
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
	if _, err := (Seed{}).HTTPEndpoint("/x"); err == nil {
		t.Fatal("HTTPEndpoint on addressless seed did not fail")
	}
}

func TestSeedIsAddressable(t *testing.T) {
	if (Seed{}).IsAddressable() {
		t.Fatal("empty seed reported addressable")
	}
	if !addressableSeed(t).IsAddressable() {
		t.Fatal("seed with primary address not addressable")
	}
	host, _ := ParseHost("2001:db8::1")
	viaAdditional := Seed{AdditionalAddresses: Some([]Host{host})}
	if !viaAdditional.IsAddressable() {
		t.Fatal("seed with additional address not addressable")
	}
}
