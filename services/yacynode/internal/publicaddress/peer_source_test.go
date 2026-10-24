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

func sourceAfterReports(t *testing.T, reportsPerRound ...[]string) publicaddress.Source {
	t.Helper()

	source := publicaddress.For(yacymodel.None[yacymodel.Host]())
	for _, reportedHosts := range reportsPerRound {
		source.ObserveReportedHosts(context.Background(), hostsOf(t, reportedHosts...))
	}

	return source
}

func assertHost(t *testing.T, source publicaddress.Source, want string) {
	t.Helper()

	host, present := source.Host(context.Background()).Get()
	if !present || host.String() != want {
		t.Fatalf("Host = %v (present %t), want %s", host, present, want)
	}
}

func assertNoHost(t *testing.T, source publicaddress.Source) {
	t.Helper()

	if host, present := source.Host(context.Background()).Get(); present {
		t.Fatalf("Host = %v, want none", host)
	}
}

func TestPeerSourceGivesTheHostThatAMajorityOfPeersReport(t *testing.T) {
	assertHost(t, sourceAfterReports(t,
		[]string{"203.0.113.9", "203.0.113.9", "203.0.113.9", "198.51.100.4", "198.51.100.5"},
	), "203.0.113.9")
}

func TestPeerSourceIgnoresAHostThatTooFewPeersReport(t *testing.T) {
	assertNoHost(t, sourceAfterReports(t, []string{"203.0.113.9", "203.0.113.9"}))
}

func TestPeerSourceIgnoresAHostWithoutAMajority(t *testing.T) {
	assertNoHost(t, sourceAfterReports(t, []string{
		"203.0.113.9", "203.0.113.9", "203.0.113.9",
		"198.51.100.4", "198.51.100.4", "198.51.100.4",
	}))
}

func TestPeerSourceKeepsTheHostWhenPeersStopAgreeing(t *testing.T) {
	assertHost(t, sourceAfterReports(t,
		[]string{"203.0.113.9", "203.0.113.9", "203.0.113.9"},
		[]string{"198.51.100.4"},
	), "203.0.113.9")
}

func TestPeerSourceGivesANewHostThatAMajorityOfPeersReport(t *testing.T) {
	assertHost(t, sourceAfterReports(t,
		[]string{"203.0.113.9", "203.0.113.9", "203.0.113.9"},
		[]string{"198.51.100.4", "198.51.100.4", "198.51.100.4"},
	), "198.51.100.4")
}

func TestPeerSourceKeepsTheHostThatPeersReportAgain(t *testing.T) {
	sameHosts := []string{"203.0.113.9", "203.0.113.9", "203.0.113.9"}

	assertHost(t, sourceAfterReports(t, sameHosts, sameHosts), "203.0.113.9")
}

func TestPeerSourceIgnoresHostsThatPeersCannotReach(t *testing.T) {
	assertHost(t, sourceAfterReports(t, []string{
		"127.0.0.1", "127.0.0.1", "127.0.0.1",
		"0.0.0.0", "fe80::1", "224.0.0.1", "peer.example",
		"203.0.113.9", "203.0.113.9", "203.0.113.9",
	}), "203.0.113.9")
}
