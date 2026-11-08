package yacypeerassembly

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionrequests"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/hostaddresses"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/processnetwork"
)

const (
	companionsDirectoryName = "companions"
	pairingSecretLifetime   = time.Minute
	pairingWait             = 30 * time.Second
)

type companions struct {
	devices     *paireddevices.Devices
	offers      *pairingoffers.Offers
	certificate tls.Certificate
}

func startBroker(cfg PeerConfig) (*natsbroker.Broker, *companions, error) {
	if !cfg.pairsCompanions() {
		broker, err := natsbroker.Start(cfg.directoryOf(brokerDirectoryName))
		return broker, nil, err
	}
	parts, err := openCompanions(cfg.directoryOf(companionsDirectoryName))
	if err != nil {
		return nil, nil, err
	}
	broker, err := natsbroker.StartListening(
		cfg.directoryOf(brokerDirectoryName),
		natsbroker.Listening{
			Authentication: companionaccess.New(
				parts.devices,
				parts.offers,
				wallclock.Clock{},
				pageServicePermissions(),
				pairingWait,
			),
			Address:     cfg.CompanionAddr,
			Certificate: parts.certificate,
		},
	)
	if err != nil {
		return nil, nil, errors.Join(err, parts.devices.Close())
	}
	return broker, parts, nil
}

func openCompanions(directory string) (*companions, error) {
	certificate, err := brokerkey.Settle(directory)
	if err != nil {
		return nil, err
	}
	devices, err := paireddevices.Open(directory)
	if err != nil {
		return nil, err
	}
	return &companions{
		devices:     devices,
		offers:      pairingoffers.New(wallclock.Clock{}, pairingSecretLifetime, pairingWait),
		certificate: certificate,
	}, nil
}

func pageServicePermissions() natspermissions.Permissions {
	return natspermissions.JoinedFrom(
		pagescrapeassembly.ServiceConfig{
			ScrapeRequestDurable: pagescrapeassembly.DefaultScrapeRequestDurable,
		}.BrokerPermissions(),
		pagerwiassembly.ServiceConfig{
			PageOfferDurable: pagerwiassembly.DefaultPageOfferDurable,
		}.BrokerPermissions(),
	)
}

func (c *companions) register(
	mux *http.ServeMux,
	broker *natsbroker.Broker,
	tabBar webuistyle.TabBar,
) {
	if c == nil {
		return
	}
	companionspage.Register(mux, companionspage.Collaborators{
		Devices:   c.devices,
		Unpairing: companionaccess.NewUnpairing(c.devices, broker),
		Offers:    offerOpening{offers: c.offers},
		Broker:    broker,
	}, c.pairLinkFor(broker), tabBar)
}

func (c *companions) pairLinkFor(broker *natsbroker.Broker) companionprotocol.PairLink {
	address, _ := broker.ListeningAddress().(*net.TCPAddr)
	return companionprotocol.PairLink{
		Addresses:     hostaddresses.ThisHost().AddressesFor(address),
		BrokerKeyHash: c.brokerKeyHash(),
	}
}

func (c *companions) brokerKeyHash() string {
	return companionprotocol.BrokerKeyHashOf(c.certificate.Leaf.RawSubjectPublicKeyInfo)
}

func (c *companions) servers(
	broker *natsbroker.Broker,
	natsURL string,
	network *processnetwork.Network,
) []func(context.Context) error {
	if c == nil {
		return nil
	}
	pairingRequests := companionrequests.NewPairingRequests(
		c.offers, c.devices, wallclock.Clock{}, c.brokerKeyHash(),
	)
	deviceRequests := companionrequests.NewDeviceRequests(
		c.devices,
		companionaccess.NewUnpairing(c.devices, broker),
		companionprotocol.Settings{
			PartitionExponent: pagerwiassembly.DefaultPageRWIPartitionExponent,
		},
	)
	return []func(context.Context) error{
		servedOn(natsURL, network, pairingRequests.Serve),
		servedOn(natsURL, network, deviceRequests.Serve),
	}
}

func servedOn(
	natsURL string,
	network *processnetwork.Network,
	serve func(context.Context, *nats.Conn) error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		_, connection, err := jetstreamconnect.Open(natsURL, network.DialContext)
		if err != nil {
			return err
		}
		defer connection.Close()
		return serve(ctx, connection)
	}
}

func closeCompanions(ctx context.Context, parts *companions) {
	if parts == nil {
		return
	}
	if err := parts.devices.Close(); err != nil {
		slog.WarnContext(ctx, "paired devices did not close", slog.Any("error", err))
	}
}

type offerOpening struct {
	offers *pairingoffers.Offers
}

func (o offerOpening) Open() companionspage.PairingOffer {
	return o.offers.Open()
}
