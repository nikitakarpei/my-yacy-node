package peerannouncement

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type greetResult struct {
	ObservedAddress  string
	ObservedPeerType yacymodel.Optional[yacymodel.PeerType]
	RespondingSeed   yacymodel.Seed
	KnownSeeds       []yacymodel.Seed
}

type httpPeerGreeter struct {
	exchange    peerwire.MessageExchange
	networkName string
}

func newHTTPPeerGreeter(client *http.Client, networkName string) httpPeerGreeter {
	return httpPeerGreeter{
		exchange:    peerwire.NewMessageExchange(client),
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
		ObservedAddress:  parsed.YourIP,
		ObservedPeerType: parsed.YourType,
		RespondingSeed:   respondingSeed,
		KnownSeeds:       parsed.KnownSeeds(),
	}, nil
}
