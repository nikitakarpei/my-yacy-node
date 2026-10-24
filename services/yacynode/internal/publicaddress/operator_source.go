package publicaddress

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type operatorSource struct {
	statedHost yacymodel.Host
}

func (s operatorSource) Host(context.Context) yacymodel.Optional[yacymodel.Host] {
	return yacymodel.Some(s.statedHost)
}

func (operatorSource) ObserveReportedHosts(context.Context, []yacymodel.Host) {}
