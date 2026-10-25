package peerannouncement

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

type contactReply struct {
	outcome         ContactOutcome
	reportedHost    yacymodel.Optional[yacymodel.Host]
	reachableScheme yacymodel.Optional[peerwire.Scheme]
}

func reportedHostsOf(contactReplies []contactReply) []yacymodel.Host {
	var reportedHosts []yacymodel.Host
	for _, reply := range contactReplies {
		if host, reported := reply.reportedHost.Get(); reported {
			reportedHosts = append(reportedHosts, host)
		}
	}

	return reportedHosts
}

func amountOfReachablePeersPerScheme(contactReplies []contactReply) map[peerwire.Scheme]int {
	amounts := map[peerwire.Scheme]int{peerwire.HTTP: 0, peerwire.HTTPS: 0}
	for _, reply := range contactReplies {
		if scheme, confirmed := reply.reachableScheme.Get(); confirmed {
			amounts[scheme]++
		}
	}

	return amounts
}
