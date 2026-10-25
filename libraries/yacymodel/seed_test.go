package yacymodel_test

import (
	"reflect"
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
	return yacymodel.Seed{PrimaryHost: yacymodel.Some(host), Port: yacymodel.Some(port)}
}

func hostOf(t *testing.T, s string) yacymodel.Host {
	t.Helper()
	host, err := yacymodel.ParseHost(s)
	if err != nil {
		t.Fatal(err)
	}
	return host
}

func TestSeedAddressesListPrimaryThenAdditionalHostsOnce(t *testing.T) {
	seed := addressableSeed(t)
	seed.AdditionalHosts = yacymodel.Some([]yacymodel.Host{
		hostOf(t, "2001:db8::1"),
		hostOf(t, "192.0.2.1"),
		hostOf(t, "peer.example"),
	})

	addresses, ok := seed.Addresses()
	if !ok {
		t.Fatal("Addresses absent for an addressable seed")
	}
	want := []yacymodel.Host{
		hostOf(t, "192.0.2.1"),
		hostOf(t, "2001:db8::1"),
		hostOf(t, "peer.example"),
	}
	if !reflect.DeepEqual(addresses.Hosts, want) {
		t.Errorf("Hosts = %v, want %v", addresses.Hosts, want)
	}
	if addresses.Port != yacymodel.Port(8090) {
		t.Errorf("Port = %v, want 8090", addresses.Port)
	}
}

func TestSeedAddressesReachAPeerWithoutPrimaryHost(t *testing.T) {
	seed := yacymodel.Seed{
		AdditionalHosts: yacymodel.Some([]yacymodel.Host{hostOf(t, "2001:db8::1")}),
		Port:            yacymodel.Some(yacymodel.Port(8090)),
	}

	addresses, ok := seed.Addresses()
	if !ok || !reflect.DeepEqual(addresses.Hosts, []yacymodel.Host{hostOf(t, "2001:db8::1")}) {
		t.Fatalf("Addresses = %v, %v", addresses, ok)
	}
}

func TestSeedAddressesAbsentWithoutPortOrHost(t *testing.T) {
	withoutPort := yacymodel.Seed{PrimaryHost: yacymodel.Some(hostOf(t, "192.0.2.1"))}
	if _, ok := withoutPort.Addresses(); ok {
		t.Error("Addresses present without port")
	}
	withPortZero := addressableSeed(t)
	withPortZero.Port = yacymodel.Some(yacymodel.Port(0))
	if _, ok := withPortZero.Addresses(); ok {
		t.Error("Addresses present with port zero")
	}
	withoutHost := yacymodel.Seed{Port: yacymodel.Some(yacymodel.Port(8090))}
	if _, ok := withoutHost.Addresses(); ok {
		t.Error("Addresses present without host")
	}
}

func TestSeedAddressesCarryTheSecurePortOnlyWhenTheSeedOffersSSL(t *testing.T) {
	for name, scenario := range map[string]struct {
		capabilities yacymodel.Optional[yacymodel.PeerCapabilities]
		want         yacymodel.Optional[yacymodel.Port]
	}{
		"ssl offered":          {yacymodel.Some(yacymodel.PeerCapabilities{SSLAvailable: true}), yacymodel.Some(yacymodel.Port(8443))},
		"ssl not offered":      {yacymodel.Some(yacymodel.PeerCapabilities{}), yacymodel.None[yacymodel.Port]()},
		"capabilities unknown": {yacymodel.None[yacymodel.PeerCapabilities](), yacymodel.None[yacymodel.Port]()},
	} {
		t.Run(name, func(t *testing.T) {
			seed := addressableSeed(t)
			seed.SecurePort = yacymodel.Some(yacymodel.Port(8443))
			seed.Capabilities = scenario.capabilities

			addresses, _ := seed.Addresses()
			if addresses.SecurePort != scenario.want {
				t.Errorf("SecurePort = %v, want %v", addresses.SecurePort, scenario.want)
			}
		})
	}
}

func TestSeedIsAddressable(t *testing.T) {
	if (yacymodel.Seed{}).IsAddressable() {
		t.Fatal("empty seed reported addressable")
	}
	if !addressableSeed(t).IsAddressable() {
		t.Fatal("seed with primary host not addressable")
	}
	host, _ := yacymodel.ParseHost("2001:db8::1")
	viaAdditional := addressableSeed(t)
	viaAdditional.PrimaryHost = yacymodel.None[yacymodel.Host]()
	viaAdditional.AdditionalHosts = yacymodel.Some([]yacymodel.Host{host})
	if !viaAdditional.IsAddressable() {
		t.Fatal("seed with additional host not addressable")
	}
	withoutPort := addressableSeed(t)
	withoutPort.Port = yacymodel.None[yacymodel.Port]()
	if withoutPort.IsAddressable() {
		t.Fatal("seed without port reported addressable")
	}
}

func TestSeedAcceptsRemoteIndexOnlyWhenItsCapabilitiesSaySo(t *testing.T) {
	for name, scenario := range map[string]struct {
		seed        yacymodel.Seed
		wantAccepts bool
	}{
		"unknown capabilities": {seed: yacymodel.Seed{}},
		"refuses remote index": {
			seed: yacymodel.Seed{Capabilities: yacymodel.Some(yacymodel.PeerCapabilities{})},
		},
		"accepts remote index": {
			seed: yacymodel.Seed{
				Capabilities: yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true}),
			},
			wantAccepts: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := scenario.seed.AcceptsRemoteIndex(); got != scenario.wantAccepts {
				t.Fatalf("AcceptsRemoteIndex = %v, want %v", got, scenario.wantAccepts)
			}
		})
	}
}
