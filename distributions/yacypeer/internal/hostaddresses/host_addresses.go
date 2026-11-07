// Package hostaddresses lists the addresses of this host at which a device can try to reach a
// listener of the peer, most likely first: the address of the default route, then addresses on
// network cards and on bridges that hold one, then VPN interfaces, then every other interface.
// Container and VM bridges come last, so a short list leaves them out.
package hostaddresses

import (
	"cmp"
	"context"
	"io/fs"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	maxAddresses       = 6
	probeAddr          = "192.0.2.1:9"
	interfaceClassRoot = "/sys/class/net"
	wireguardDevice    = "DEVTYPE=wireguard"
)

type likelihood int

const (
	defaultRoute likelihood = iota
	networkCard
	vpnInterface
	otherInterface
)

type Interface struct {
	Name string
	IPs  []net.IP
}

type Host struct {
	classes    fs.FS
	outgoingIP net.IP
	interfaces []Interface
}

type candidate struct {
	ip         net.IP
	likelihood likelihood
}

func HostOf(interfaces []Interface, classes fs.FS, outgoingIP net.IP) Host {
	return Host{interfaces: interfaces, classes: classes, outgoingIP: outgoingIP}
}

func ThisHost() Host {
	return HostOf(interfacesOfThisHost(), os.DirFS(interfaceClassRoot), outgoingIP())
}

func interfacesOfThisHost() []Interface {
	found, err := net.Interfaces()
	if err != nil {
		return nil
	}
	interfaces := make([]Interface, 0, len(found))
	for _, each := range found {
		if each.Flags&net.FlagUp == 0 {
			continue
		}
		addresses, err := each.Addrs()
		if err != nil {
			continue
		}
		ips := make([]net.IP, 0, len(addresses))
		for _, address := range addresses {
			if network, isNetwork := address.(*net.IPNet); isNetwork {
				ips = append(ips, network.IP)
			}
		}
		interfaces = append(interfaces, Interface{Name: each.Name, IPs: ips})
	}
	return interfaces
}

func outgoingIP() net.IP {
	probe, err := (&net.Dialer{}).DialContext(context.Background(), "udp", probeAddr)
	if err != nil {
		return nil
	}
	defer func() { _ = probe.Close() }()
	return probe.LocalAddr().(*net.UDPAddr).IP
}

func (h Host) AddressesFor(listener *net.TCPAddr) []string {
	port := strconv.Itoa(listener.Port)
	if !listener.IP.IsUnspecified() {
		return []string{net.JoinHostPort(listener.IP.String(), port)}
	}
	candidates := h.candidates()
	slices.SortStableFunc(candidates, func(a, b candidate) int {
		return cmp.Or(
			cmp.Compare(a.likelihood, b.likelihood),
			cmp.Compare(familyOf(a.ip), familyOf(b.ip)),
		)
	})
	addresses := make([]string, 0, min(len(candidates), maxAddresses))
	for _, each := range candidates[:min(len(candidates), maxAddresses)] {
		addresses = append(addresses, net.JoinHostPort(each.ip.String(), port))
	}
	return addresses
}

func (h Host) candidates() []candidate {
	var candidates []candidate
	for _, each := range h.interfaces {
		kind := h.likelihoodOf(each.Name)
		for _, ip := range each.IPs {
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsMulticast() ||
				ip.IsUnspecified() {
				continue
			}
			if ip.Equal(h.outgoingIP) {
				candidates = append(candidates, candidate{ip: ip, likelihood: defaultRoute})
				continue
			}
			candidates = append(candidates, candidate{ip: ip, likelihood: kind})
		}
	}
	return candidates
}

func (h Host) likelihoodOf(name string) likelihood {
	switch {
	case h.isNetworkCard(name) || h.holdsNetworkCard(name):
		return networkCard
	case h.isVPN(name):
		return vpnInterface
	default:
		return otherInterface
	}
}

func (h Host) isNetworkCard(name string) bool {
	_, err := fs.Stat(h.classes, name+"/device")
	return err == nil
}

func (h Host) holdsNetworkCard(name string) bool {
	ports, err := fs.ReadDir(h.classes, name+"/brif")
	if err != nil {
		return false
	}
	return slices.ContainsFunc(ports, func(port fs.DirEntry) bool {
		return h.isNetworkCard(port.Name())
	})
}

func (h Host) isVPN(name string) bool {
	if _, err := fs.Stat(h.classes, name+"/tun_flags"); err == nil {
		return true
	}
	uevent, err := fs.ReadFile(h.classes, name+"/uevent")
	return err == nil && strings.Contains(string(uevent), wireguardDevice)
}

func familyOf(ip net.IP) int {
	if ip.To4() != nil {
		return 4
	}
	return 6
}
