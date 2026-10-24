package peerannouncement

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type contactReply struct {
	outcome      ContactOutcome
	observedHost yacymodel.Optional[yacymodel.Host]
}

func outcomesOf(contactReplies []contactReply) []ContactOutcome {
	outcomes := make([]ContactOutcome, 0, len(contactReplies))
	for _, reply := range contactReplies {
		outcomes = append(outcomes, reply.outcome)
	}

	return outcomes
}

func observedHostsOf(contactReplies []contactReply) []yacymodel.Host {
	var observedHosts []yacymodel.Host
	for _, reply := range contactReplies {
		if host, observed := reply.observedHost.Get(); observed {
			observedHosts = append(observedHosts, host)
		}
	}

	return observedHosts
}
