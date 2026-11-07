package hostaddresses_test

import (
	"net"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/hostaddresses"
)

const port = 4223

var everyAddress = &net.TCPAddr{IP: net.IPv4zero, Port: port}

var interfaceClasses = fstest.MapFS{
	"wlp1s0/device":        {},
	"enp3s0/device":        {},
	"br0/brif/enp3s0":      {},
	"docker0/brif/veth123": {},
	"tailscale0/tun_flags": {},
	"wg0/uevent":           {Data: []byte("DEVTYPE=wireguard\nINTERFACE=wg0\n")},
}

func on(name string, ip string) hostaddresses.Interface {
	return hostaddresses.Interface{Name: name, IPs: []net.IP{net.ParseIP(ip)}}
}

func TestAListenerOnOneAddressIsReachedThereOnly(t *testing.T) {
	host := hostaddresses.HostOf(
		[]hostaddresses.Interface{on("wlp1s0", "192.168.10.157")},
		interfaceClasses,
		nil,
	)

	addresses := host.AddressesFor(&net.TCPAddr{IP: net.ParseIP("100.101.102.103"), Port: port})

	if want := []string{"100.101.102.103:4223"}; !reflect.DeepEqual(addresses, want) {
		t.Errorf("addresses = %v, want %v", addresses, want)
	}
}

func TestTheAddressesOfAListenerOnEveryAddressComeMostLikelyFirst(t *testing.T) {
	host := hostaddresses.HostOf([]hostaddresses.Interface{
		on("docker0", "172.17.0.1"),
		on("lo", "127.0.0.1"),
		on("wg0", "10.8.0.2"),
		on("tailscale0", "100.101.102.103"),
		on("wlp1s0", "2001:db8::7"),
		on("br0", "10.0.0.5"),
		on("wlp1s0", "fe80::1"),
		on("wlp1s0", "192.168.10.157"),
	}, interfaceClasses, net.ParseIP("192.168.10.157"))

	addresses := host.AddressesFor(everyAddress)

	want := []string{
		"192.168.10.157:4223",
		"10.0.0.5:4223",
		"[2001:db8::7]:4223",
		"10.8.0.2:4223",
		"100.101.102.103:4223",
		"172.17.0.1:4223",
	}
	if !reflect.DeepEqual(addresses, want) {
		t.Errorf("addresses = %v, want %v", addresses, want)
	}
}

func TestAShortListLeavesTheLeastLikelyAddressesOut(t *testing.T) {
	host := hostaddresses.HostOf([]hostaddresses.Interface{
		on("docker0", "172.17.0.1"),
		on("wlp1s0", "192.168.10.157"),
		on("wlp1s0", "192.168.10.158"),
		on("wlp1s0", "192.168.10.159"),
		on("wlp1s0", "192.168.10.160"),
		on("wlp1s0", "192.168.10.161"),
		on("wlp1s0", "192.168.10.162"),
	}, interfaceClasses, nil)

	addresses := host.AddressesFor(everyAddress)

	if len(addresses) != 6 || addresses[5] == "172.17.0.1:4223" {
		t.Errorf("addresses = %v, want six without the container bridge", addresses)
	}
}

func TestThisHostListsAtLeastOneAddress(t *testing.T) {
	if addresses := hostaddresses.ThisHost().AddressesFor(everyAddress); len(addresses) == 0 {
		t.Error("this host lists no address")
	}
}
