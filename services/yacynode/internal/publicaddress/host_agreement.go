package publicaddress

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const minimumAgreeingPeers = 3

type hostAgreement struct {
	host           yacymodel.Host
	agreeingPeers  int
	reportingPeers int
}

func agreementAmong(reportedHosts []yacymodel.Host) hostAgreement {
	agreeingPeersPerHost := make(map[yacymodel.Host]int)
	var agreement hostAgreement
	for _, host := range reportedHosts {
		if !isAdvertisable(host) {
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

func isAdvertisable(host yacymodel.Host) bool {
	addr, isAddr := host.Addr()

	return isAddr && addr.IsGlobalUnicast() && !yacymodel.IsLocalNetworkAddress(addr)
}

func (a hostAgreement) reached() bool {
	return a.agreeingPeers >= minimumAgreeingPeers && 2*a.agreeingPeers > a.reportingPeers
}
