package publicaddress

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type statedAddress struct {
	host yacymodel.Host
}

func (a statedAddress) Host(context.Context) yacymodel.Optional[yacymodel.Host] {
	return yacymodel.Some(a.host)
}

func (statedAddress) Settle(context.Context, []yacymodel.Host) bool {
	return false
}
