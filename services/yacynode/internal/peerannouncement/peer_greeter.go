package peerannouncement

import (
	"context"
	"net/netip"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peercontact"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type greetResult struct {
	Origin           peerwire.Origin
	ReportedHost     yacymodel.Optional[yacymodel.Host]
	ObservedPeerType yacymodel.Optional[yacymodel.PeerType]
	RespondingSeed   yacymodel.Seed
	KnownSeeds       []yacymodel.Seed
}

type peerGreeter struct {
	contacts    *peercontact.Contacts
	networkName string
}

func newPeerGreeter(contacts *peercontact.Contacts, networkName string) peerGreeter {
	return peerGreeter{contacts: contacts, networkName: networkName}
}

func (g peerGreeter) Greet(
	ctx context.Context,
	peer yacymodel.Hash,
	addresses yacymodel.PeerAddresses,
	self yacymodel.Seed,
	count int,
) (greetResult, bool) {
	request := yacyproto.HelloRequest{
		NetworkName: g.networkName,
		Seed:        self,
		Count:       count,
		Iam:         self.Hash,
	}

	answer, answered := g.contacts.Greet(ctx, addresses, peercontact.Question{
		Request: peerwire.Request{Peer: peer, Path: yacyproto.PathHello, Form: request.Form()},
		Accepts: func(fields yacyproto.ResponseFields) bool { return isHelloOf(ctx, peer, fields) },
	})
	if !answered {
		return greetResult{}, false
	}
	hello, _ := yacyproto.ParseHelloResponse(ctx, answer.Fields)
	respondingSeed, _ := hello.OwnSeed().Get()

	return greetResult{
		Origin:           answer.Origin,
		ReportedHost:     reportedHostFrom(hello.YourIP),
		ObservedPeerType: hello.YourType,
		RespondingSeed:   respondingSeed,
		KnownSeeds:       hello.KnownSeeds(),
	}, true
}

func isHelloOf(ctx context.Context, peer yacymodel.Hash, fields yacyproto.ResponseFields) bool {
	hello, err := yacyproto.ParseHelloResponse(ctx, fields)
	if err != nil {
		return false
	}
	respondingSeed, found := hello.OwnSeed().Get()
	_, addressable := respondingSeed.Addresses()

	return found && addressable && respondingSeed.Hash == peer
}

func reportedHostFrom(yourIP string) yacymodel.Optional[yacymodel.Host] {
	addr, err := netip.ParseAddr(yourIP)
	if err != nil {
		return yacymodel.None[yacymodel.Host]()
	}

	return yacymodel.Some(yacymodel.HostFrom(addr.Unmap()))
}
