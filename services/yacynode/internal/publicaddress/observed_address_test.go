package publicaddress_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/publicaddress"
)

func hostsOf(t *testing.T, rawHosts ...string) []yacymodel.Host {
	t.Helper()

	hosts := make([]yacymodel.Host, 0, len(rawHosts))
	for _, rawHost := range rawHosts {
		host, err := yacymodel.ParseHost(rawHost)
		if err != nil {
			t.Fatalf("parse host %q: %v", rawHost, err)
		}
		hosts = append(hosts, host)
	}

	return hosts
}

func assertHost(t *testing.T, address publicaddress.PublicAddress, want string) {
	t.Helper()

	host, present := address.Host(context.Background()).Get()
	if !present || host.String() != want {
		t.Fatalf("Host = %v (present %t), want %s", host, present, want)
	}
}

func assertNoHost(t *testing.T, address publicaddress.PublicAddress) {
	t.Helper()

	if host, present := address.Host(context.Background()).Get(); present {
		t.Fatalf("Host = %v, want none", host)
	}
}

func TestSettleAdoptsTheHostThatAMajorityOfPeersReport(t *testing.T) {
	address := publicaddress.For(yacymodel.None[yacymodel.Host]())

	changed := address.Settle(context.Background(), hostsOf(t,
		"203.0.113.9", "203.0.113.9", "203.0.113.9", "198.51.100.4", "198.51.100.5",
	))

	if !changed {
		t.Fatal("Settle = false, want the address changed")
	}
	assertHost(t, address, "203.0.113.9")
}

func TestSettleIgnoresAHostThatTooFewPeersReport(t *testing.T) {
	address := publicaddress.For(yacymodel.None[yacymodel.Host]())

	changed := address.Settle(context.Background(), hostsOf(t, "203.0.113.9", "203.0.113.9"))

	if changed {
		t.Fatal("Settle = true, want no change")
	}
	assertNoHost(t, address)
}

func TestSettleIgnoresAHostWithoutAMajority(t *testing.T) {
	address := publicaddress.For(yacymodel.None[yacymodel.Host]())

	address.Settle(context.Background(), hostsOf(t,
		"203.0.113.9", "203.0.113.9", "203.0.113.9",
		"198.51.100.4", "198.51.100.4", "198.51.100.4",
	))

	assertNoHost(t, address)
}

func TestSettleKeepsTheHostWhenPeersStopAgreeing(t *testing.T) {
	address := publicaddress.For(yacymodel.None[yacymodel.Host]())
	address.Settle(context.Background(), hostsOf(t, "203.0.113.9", "203.0.113.9", "203.0.113.9"))

	changed := address.Settle(context.Background(), hostsOf(t, "198.51.100.4"))

	if changed {
		t.Fatal("Settle = true, want no change")
	}
	assertHost(t, address, "203.0.113.9")
}

func TestSettleReportsNoChangeForTheSameHost(t *testing.T) {
	address := publicaddress.For(yacymodel.None[yacymodel.Host]())
	sameHosts := hostsOf(t, "203.0.113.9", "203.0.113.9", "203.0.113.9")
	address.Settle(context.Background(), sameHosts)

	if address.Settle(context.Background(), sameHosts) {
		t.Fatal("Settle = true, want no change for the same host")
	}
}

func TestSettleAdoptsANewHostThatAMajorityOfPeersReport(t *testing.T) {
	address := publicaddress.For(yacymodel.None[yacymodel.Host]())
	address.Settle(context.Background(), hostsOf(t, "203.0.113.9", "203.0.113.9", "203.0.113.9"))

	changed := address.Settle(context.Background(), hostsOf(t,
		"198.51.100.4", "198.51.100.4", "198.51.100.4",
	))

	if !changed {
		t.Fatal("Settle = false, want the address changed")
	}
	assertHost(t, address, "198.51.100.4")
}

func TestSettleIgnoresHostsThatPeersCannotReach(t *testing.T) {
	address := publicaddress.For(yacymodel.None[yacymodel.Host]())

	address.Settle(context.Background(), hostsOf(t,
		"127.0.0.1", "127.0.0.1", "127.0.0.1",
		"0.0.0.0", "fe80::1", "224.0.0.1", "peer.example",
		"203.0.113.9", "203.0.113.9", "203.0.113.9",
	))

	assertHost(t, address, "203.0.113.9")
}
