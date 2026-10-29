package peerannouncement

import (
	"context"
	"net/netip"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const greetBudget = 30 * time.Second

type greetResult struct {
	BaseURL          peerwire.BaseURL
	ReportedHost     yacymodel.Optional[yacymodel.Host]
	ObservedPeerType yacymodel.Optional[yacymodel.PeerType]
	RespondingSeed   yacymodel.Seed
	KnownSeeds       []yacymodel.Seed
}

type peerGreeter struct {
	walker      peerwire.Walker
	networkName string
}

func newPeerGreeter(walker peerwire.Walker, networkName string) peerGreeter {
	return peerGreeter{walker: walker, networkName: networkName}
}

func (g peerGreeter) Greet(
	ctx context.Context,
	peer yacymodel.Hash,
	record peerroster.PeerRecord,
	self yacymodel.Seed,
	count int,
) (greetResult, bool) {
	request := yacyproto.HelloRequest{
		NetworkName: g.networkName,
		Seed:        self,
		Count:       count,
		Iam:         self.Hash,
	}

	answer, answered := g.walker.TryInTurn(
		ctx,
		peerwire.Request{
			Callee: peerwire.Callee{Hash: peer, Identity: record.Identity},
			Path:   yacyproto.PathHello,
			Form:   request.Form(),
		},
		peerwire.BaseURLsFrom(record.Addresses, record.Identity),
		func(fields yacyproto.ResponseFields) bool { return isHelloOf(ctx, peer, fields) },
		greetBudget,
	)
	if !answered {
		return greetResult{}, false
	}
	hello, _ := yacyproto.ParseHelloResponse(ctx, answer.Fields)
	respondingSeed, _ := hello.OwnSeed().Get()

	return greetResult{
		BaseURL:          answer.BaseURL,
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
