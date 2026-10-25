package peeradmission

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peercontact"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const backPingsAtOnce = 16

type callerBackPing struct {
	contacts *peercontact.Contacts
	slots    chan struct{}
}

func newCallerBackPing(contacts *peercontact.Contacts) callerBackPing {
	return callerBackPing{contacts: contacts, slots: make(chan struct{}, backPingsAtOnce)}
}

var _ callerReachabilityProbe = callerBackPing{}

func (p callerBackPing) IsReachable(
	ctx context.Context,
	caller yacymodel.Hash,
	addresses yacymodel.PeerAddresses,
	self yacymodel.Hash,
	networkName string,
) bool {
	select {
	case p.slots <- struct{}{}:
		defer func() { <-p.slots }()
	default:
		slog.WarnContext(ctx, "caller back-ping skipped, too many back-pings running",
			slog.String("peer", caller.String()),
		)

		return false
	}

	query := yacyproto.QueryRequest{
		NetworkName: networkName,
		YouAre:      caller,
		Iam:         self,
		Object:      yacyproto.ObjectRWICount,
	}
	_, answered := p.contacts.Probe(ctx, addresses, peercontact.Question{
		Request: peerwire.Request{
			Peer: caller,
			Path: yacyproto.PathQuery,
			Form: query.Form(),
		},
		Accepts: isRWICountReply,
	})

	return answered
}

func isRWICountReply(fields yacyproto.ResponseFields) bool {
	if fields[yacyproto.FieldResponse] == "" {
		return false
	}
	_, err := yacyproto.ParseQueryResponse(fields)

	return err == nil
}
