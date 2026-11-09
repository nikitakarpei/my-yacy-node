package companionrequests

import (
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

var (
	errOlderProtocol = errors.New("the app speaks an older protocol")
	errNewerProtocol = errors.New("the app speaks a newer protocol")
)

type requestRefusal struct {
	reason companionprotocol.RefusalReason
	cause  error
}

func protocolRefusalOf(protocolVersion int) *requestRefusal {
	switch {
	case protocolVersion < companionprotocol.ProtocolVersion:
		return &requestRefusal{reason: companionprotocol.ReasonAppOutdated, cause: errOlderProtocol}
	case protocolVersion > companionprotocol.ProtocolVersion:
		return &requestRefusal{
			reason: companionprotocol.ReasonPeerOutdated,
			cause:  errNewerProtocol,
		}
	}
	return nil
}
