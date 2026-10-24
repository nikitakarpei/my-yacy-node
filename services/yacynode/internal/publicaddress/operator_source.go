package publicaddress

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type OperatorSource struct {
	statedHost yacymodel.Host
}

func Stated(host yacymodel.Host) OperatorSource {
	return OperatorSource{statedHost: host}
}

func (s OperatorSource) Host(context.Context) yacymodel.Optional[yacymodel.Host] {
	return yacymodel.Some(s.statedHost)
}
