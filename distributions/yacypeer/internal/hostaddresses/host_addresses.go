// Package hostaddresses lists the addresses of a host at which a device can try to reach a
// listener of the peer, most likely first: the address of the default route, then addresses on
// network cards and on bridges that hold one, then VPN interfaces, then every other interface.
// Container and VM bridges come last, so a short list leaves them out.
package hostaddresses

import (
	"cmp"
	"io/fs"
	"net"
	"slices"
	"strconv"
	"strings"
)

const (
	maxAddresses    = 6
	wireguardDevice = "DEVTYPE=wireguard"
)

type interfaceKind int

const (
	defaultRoute interfaceKind = iota
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
	ip   net.IP
	kind interfaceKind
}

func HostFrom(interfaces []Interface, classes fs.FS, outgoingIP net.IP) Host {
	return Host{interfaces: interfaces, classes: classes, outgoingIP: outgoingIP}
}

func (h Host) AddressesFor(listener *net.TCPAddr) []string {
	port := strconv.Itoa(listener.Port)
	if !listener.IP.IsUnspecified() {
		return []string{net.JoinHostPort(listener.IP.String(), port)}
	}
	candidates := h.candidates()
	slices.SortStableFunc(candidates, func(a, b candidate) int {
		return cmp.Or(
			cmp.Compare(a.kind, b.kind),
			cmp.Compare(familyOf(a.ip), familyOf(b.ip)),
		)
	})
	addresses := make([]string, 0, min(len(candidates), maxAddresses))
	for _, candidate := range candidates[:min(len(candidates), maxAddresses)] {
		addresses = append(addresses, net.JoinHostPort(candidate.ip.String(), port))
	}
	return addresses
}

func (h Host) candidates() []candidate {
	var candidates []candidate
	for _, networkInterface := range h.interfaces {
		kind := h.kindOf(networkInterface.Name)
		for _, ip := range networkInterface.IPs {
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsMulticast() ||
				ip.IsUnspecified() {
				continue
			}
			if ip.Equal(h.outgoingIP) {
				candidates = append(candidates, candidate{ip: ip, kind: defaultRoute})
				continue
			}
			candidates = append(candidates, candidate{ip: ip, kind: kind})
		}
	}
	return candidates
}

func (h Host) kindOf(name string) interfaceKind {
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
