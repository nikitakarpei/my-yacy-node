package yacypeerassembly

import (
	"context"
	"errors"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	yacynodeassembly "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/connecttimers/tcp"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/ipv6stack"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/jointlistener"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilnode"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerlist"
)

type yggdrasilOverlay struct {
	node       *yggdrasilnode.Node
	stack      *ipv6stack.Stack
	peerChoice *yggdrasilpeerchoice.Choice
}

func yggdrasilOverlayFor(ctx context.Context, cfg PeerConfig) (yggdrasilOverlay, error) {
	if !cfg.joinsYggdrasil() {
		return yggdrasilOverlay{}, nil
	}
	key, err := yggdrasilkey.Settle(cfg.directoryOf(yggdrasilnode.Name))
	if err != nil {
		return yggdrasilOverlay{}, err
	}
	node, err := yggdrasilnode.Start(
		servicelog.ContextFor(ctx, yggdrasilnode.Name),
		key,
		cfg.YggdrasilPeering,
	)
	if err != nil {
		return yggdrasilOverlay{}, err
	}
	stack, err := ipv6stack.New(node.Address(), node, node.MTU())
	if err != nil {
		node.Stop()
		return yggdrasilOverlay{}, err
	}
	return yggdrasilOverlay{node: node, stack: stack, peerChoice: peerChoiceFor(cfg, node)}, nil
}

func peerChoiceFor(cfg PeerConfig, node *yggdrasilnode.Node) *yggdrasilpeerchoice.Choice {
	if !cfg.choosesYggdrasilPeers() {
		return nil
	}
	return yggdrasilpeerchoice.New(
		yggdrasilpeerlist.New(cfg.YggdrasilPeerList),
		node,
		tcp.New(),
		wallclock.Clock{},
	)
}

func (o yggdrasilOverlay) isJoined() bool {
	return o.node != nil
}

func (o yggdrasilOverlay) egressOverlays() []egressproxy.Overlay {
	if !o.isJoined() {
		return nil
	}
	return []egressproxy.Overlay{{Addresses: yggdrasilnode.NetworkAddresses, Dialer: o.stack}}
}

func (o yggdrasilOverlay) runs() []func(context.Context) error {
	if !o.isJoined() {
		return nil
	}
	runs := []func(context.Context) error{o.stack.Run}
	if o.peerChoice != nil {
		runs = append(runs, func(ctx context.Context) error {
			return o.peerChoice.Run(servicelog.ContextFor(ctx, yggdrasilnode.Name))
		})
	}
	return runs
}

func (o yggdrasilOverlay) stop() {
	if o.isJoined() {
		o.stack.Close()
		o.node.Stop()
	}
}

func (o yggdrasilOverlay) yacynodeListenersFor(
	ctx context.Context,
	cfg yacynodeassembly.ServiceConfig,
) (yacynodeassembly.Listeners, error) {
	listeners, err := yacynodeassembly.ListenersFor(ctx, cfg)
	if err != nil || !o.isJoined() {
		return listeners, err
	}
	overlayListener, err := o.stack.Listen(cfg.Identity.AdvertisePort)
	if err != nil {
		return yacynodeassembly.Listeners{}, errors.Join(err, listeners.Close())
	}
	return yacynodeassembly.Listeners{
		PeerProtocol: jointlistener.New(listeners.PeerProtocol, overlayListener),
	}, nil
}

func (o yggdrasilOverlay) yacynodeConfigFrom(
	getenv func(string) string,
) (yacynodeassembly.ServiceConfig, error) {
	return yacynodeassembly.LoadServiceConfig(func(key string) string {
		if key == yacynodeassembly.EnvAdvertiseAdditionalHosts {
			advertisedHosts := envconfig.List(getenv, key)
			return strings.Join(append(advertisedHosts, o.node.Address().String()), ",")
		}
		return getenv(key)
	})
}
