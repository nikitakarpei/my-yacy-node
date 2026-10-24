package peerannouncement

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type contactReply struct {
	outcome      ContactOutcome
	reportedHost yacymodel.Optional[yacymodel.Host]
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
