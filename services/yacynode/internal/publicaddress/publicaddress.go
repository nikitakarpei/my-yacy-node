// Package publicaddress gives the host this node advertises in its seed. Its source is
// the operator, who states the host, or else the peers: the host that most peers of the
// last announce round report they saw this node connect from.
package publicaddress

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Source interface {
	Host(ctx context.Context) yacymodel.Optional[yacymodel.Host]
	ObserveReportedHosts(ctx context.Context, reportedHosts []yacymodel.Host)
}

func For(statedHost yacymodel.Optional[yacymodel.Host]) Source {
	if host, stated := statedHost.Get(); stated {
		return operatorSource{statedHost: host}
	}

	return &peerSource{}
}
