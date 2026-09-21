package yacymodel

import (
	"net/url"
	"strings"
)

const worldWideWebLabel = "www."

// SiteOf gives the site an address belongs to: the host in lower case, without
// the world wide web label and without a trailing dot. Two addresses that one
// site serves give one site. An address without a host gives the address.
func SiteOf(address string) string {
	readAddress, err := url.Parse(address)
	if err != nil || readAddress.Hostname() == "" {
		return address
	}

	return strings.TrimPrefix(
		strings.TrimSuffix(strings.ToLower(readAddress.Hostname()), "."),
		worldWideWebLabel,
	)
}
