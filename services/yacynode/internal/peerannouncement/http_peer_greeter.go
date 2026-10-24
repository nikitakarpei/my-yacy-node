package peerannouncement

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type greetResult struct {
	ReportedHost     yacymodel.Optional[yacymodel.Host]
	ObservedPeerType yacymodel.Optional[yacymodel.PeerType]
	RespondingSeed   yacymodel.Seed
	KnownSeeds       []yacymodel.Seed
}

type httpPeerGreeter struct {
	exchange    peerwire.FormExchange
	networkName string
}

func newHTTPPeerGreeter(client *http.Client, networkName string) httpPeerGreeter {
	return httpPeerGreeter{
		exchange:    peerwire.NewFormExchange(client),
		networkName: networkName,
	}
}

func (g httpPeerGreeter) Greet(
	ctx context.Context,
	networkAddress yacymodel.NetworkAddress,
	self yacymodel.Seed,
	count int,
) (greetResult, error) {
	request := yacyproto.HelloRequest{
		NetworkName: g.networkName,
		Seed:        self,
		Count:       count,
		Iam:         self.Hash,
	}

	msg, err := g.exchange.Exchange(
		ctx,
		networkAddress.String(),
		yacyproto.PathHello,
		request.Form(),
	)
	if err != nil {
		return greetResult{}, err
	}

	parsed, err := yacyproto.ParseHelloResponse(ctx, msg)
	if err != nil {
		return greetResult{}, fmt.Errorf("parse hello response: %w", err)
	}
	respondingSeed, found := parsed.OwnSeed().Get()
	if !found {
		return greetResult{}, fmt.Errorf("parse hello response: responding seed absent")
	}

	return greetResult{
		ReportedHost:     reportedHostFrom(parsed.YourIP),
		ObservedPeerType: parsed.YourType,
		RespondingSeed:   respondingSeed,
		KnownSeeds:       parsed.KnownSeeds(),
	}, nil
}

func reportedHostFrom(yourIP string) yacymodel.Optional[yacymodel.Host] {
	addr, err := netip.ParseAddr(yourIP)
	if err != nil {
		return yacymodel.None[yacymodel.Host]()
	}

	return yacymodel.Some(yacymodel.HostFrom(addr.Unmap()))
}
