// Package pagesite tells which site a page address belongs to.
package pagesite

import (
	"net/netip"

	"golang.org/x/net/publicsuffix"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

func RegistrableDomainOf(address canonicalurl.CanonicalURL) string {
	host := address.Hostname()
	if _, err := netip.ParseAddr(host); err == nil {
		return host
	}
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host
	}
	return domain
}
