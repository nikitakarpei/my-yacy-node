package publicaddress

import (
	"net/netip"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const minimumAgreeingPeers = 3

type hostAgreement struct {
	host           yacymodel.Host
	agreeingPeers  int
	reportingPeers int
}

func agreementAmong(observedHosts []yacymodel.Host) hostAgreement {
	agreeingPeersPerHost := make(map[yacymodel.Host]int)
	var agreement hostAgreement
	for _, observedHost := range observedHosts {
		host, advertisable := advertisableHostFrom(observedHost)
		if !advertisable {
			continue
		}
		agreement.reportingPeers++
		agreeingPeersPerHost[host]++
		if agreeingPeersPerHost[host] > agreement.agreeingPeers {
			agreement.host = host
			agreement.agreeingPeers = agreeingPeersPerHost[host]
		}
	}

	return agreement
}

func advertisableHostFrom(observedHost yacymodel.Host) (yacymodel.Host, bool) {
	addr, err := netip.ParseAddr(observedHost.String())
	if err != nil {
		return yacymodel.Host{}, false
	}
	addr = addr.Unmap()
	if addr.IsLoopback() || addr.IsUnspecified() || addr.IsMulticast() ||
		addr.IsLinkLocalUnicast() {
		return yacymodel.Host{}, false
	}
	host, err := yacymodel.ParseHost(addr.String())

	return host, err == nil
}

func (a hostAgreement) reached() bool {
	return a.agreeingPeers >= minimumAgreeingPeers && 2*a.agreeingPeers > a.reportingPeers
}
