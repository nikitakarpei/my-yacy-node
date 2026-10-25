package yacymodel

import "net/netip"

var sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")

func IsLocalNetworkAddress(address netip.Addr) bool {
	return address.IsLoopback() ||
		address.IsPrivate() ||
		address.IsLinkLocalUnicast() ||
		sharedAddressSpace.Contains(address)
}
