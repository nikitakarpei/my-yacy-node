package binaryfuse_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/welllinkedhosts/binaryfuse"
)

const mostHostsOutsideTheListHeldByMistake = 0.01

func wellLinkedHostsIn(t *testing.T, hostList string) binaryfuse.WellLinkedHosts {
	t.Helper()

	hosts, err := binaryfuse.New(strings.NewReader(hostList))
	if err != nil {
		t.Fatalf("build the well-linked hosts: %v", err)
	}

	return hosts
}

func TestAListedHostHoldsEveryAddressOfItsSite(t *testing.T) {
	t.Parallel()

	hosts := wellLinkedHostsIn(t, "www.kernel.org\nGNU.org.\n\n  debian.org  \n")

	for _, address := range []string{
		"https://www.kernel.org/doc/",
		"https://kernel.org/",
		"http://www.gnu.org/licenses/",
		"https://debian.org/",
	} {
		if !hosts.HoldsHostOf(address) {
			t.Errorf("the well-linked hosts do not hold the host of %q", address)
		}
	}
	if hosts.AmountOfHosts() != 3 {
		t.Fatalf("amount of hosts = %d, want the 3 listed", hosts.AmountOfHosts())
	}
}

func TestASubdomainOfAListedHostIsNotHeld(t *testing.T) {
	t.Parallel()

	hosts := wellLinkedHostsIn(t, "kernel.org\n")

	if hosts.HoldsHostOf("https://lore.kernel.org/") {
		t.Fatal("the well-linked hosts hold a subdomain nobody listed")
	}
}

func TestFewHostsOutsideTheListAreHeldByMistake(t *testing.T) {
	t.Parallel()

	var hostList strings.Builder
	for host := range 10000 {
		fmt.Fprintf(&hostList, "listed-%d.example\n", host)
	}
	hosts := wellLinkedHostsIn(t, hostList.String())

	amountHeldByMistake := 0
	const amountOfUnlistedHosts = 100000
	for host := range amountOfUnlistedHosts {
		if hosts.HoldsHostOf(fmt.Sprintf("https://unlisted-%d.example/", host)) {
			amountHeldByMistake++
		}
	}
	if share := float64(
		amountHeldByMistake,
	) / amountOfUnlistedHosts; share > mostHostsOutsideTheListHeldByMistake {
		t.Fatalf(
			"%.4f of the unlisted hosts are held, want at most %.2f",
			share,
			mostHostsOutsideTheListHeldByMistake,
		)
	}
}

func TestAnEmptyHostListHoldsNoHost(t *testing.T) {
	t.Parallel()

	hosts := wellLinkedHostsIn(t, "\n\n")

	if hosts.HoldsHostOf("https://kernel.org/") {
		t.Fatal("an empty host list holds a host")
	}
}

func TestAHostListThatCannotBeReadIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := binaryfuse.New(iotest.ErrReader(errors.New("disk gone"))); err == nil {
		t.Fatal("an unreadable host list built well-linked hosts, want an error")
	}
}
