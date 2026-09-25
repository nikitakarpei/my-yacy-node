package yacymodel

import (
	"net/url"
	"strings"
)

const worldWideWebLabel = "www."

func SiteOf(address string) string {
	readAddress, err := url.Parse(address)
	if err != nil || readAddress.Hostname() == "" {
		return address
	}

	return SiteOfHost(readAddress.Hostname())
}

func SiteOfHost(host string) string {
	return strings.TrimPrefix(
		strings.TrimSuffix(strings.ToLower(host), "."),
		worldWideWebLabel,
	)
}
