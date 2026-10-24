// Package publicaddress owns the host this node advertises in its seed: the host the
// operator states, or else the host that most peers of the last announce round saw this
// node connect from.
package publicaddress

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PublicAddress interface {
	Host(ctx context.Context) yacymodel.Optional[yacymodel.Host]
	Settle(ctx context.Context, observedHosts []yacymodel.Host) bool
}

func For(statedHost yacymodel.Optional[yacymodel.Host]) PublicAddress {
	if host, stated := statedHost.Get(); stated {
		return statedAddress{host: host}
	}

	return &observedAddress{}
}
