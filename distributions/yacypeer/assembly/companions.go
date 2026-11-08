package yacypeerassembly

import (
	"context"
	"crypto/tls"
	"log/slog"
	"time"

	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

const (
	companionsDirectoryName = "companions"
	pairingSecretLifetime   = time.Minute
	pairingWait             = 30 * time.Second
)

type companions interface {
	startBroker(directory string) (*natsbroker.Broker, error)
	close(ctx context.Context)
}

type noCompanions struct{}

type pairedCompanions struct {
	devices     *paireddevices.Devices
	offers      *pairingoffers.Offers
	certificate tls.Certificate
	address     string
}

func openCompanions(cfg PeerConfig) (companions, error) {
	if !cfg.pairsCompanions() {
		return noCompanions{}, nil
	}
	directory := cfg.directoryOf(companionsDirectoryName)
	certificate, err := brokerkey.Settle(directory)
	if err != nil {
		return nil, err
	}
	devices, err := paireddevices.Open(directory)
	if err != nil {
		return nil, err
	}
	return &pairedCompanions{
		devices:     devices,
		offers:      pairingoffers.New(wallclock.Clock{}, pairingSecretLifetime, pairingWait),
		certificate: certificate,
		address:     cfg.CompanionAddr,
	}, nil
}

func (noCompanions) startBroker(directory string) (*natsbroker.Broker, error) {
	return natsbroker.Start(directory)
}

func (noCompanions) close(context.Context) {}

func (c *pairedCompanions) startBroker(directory string) (*natsbroker.Broker, error) {
	return natsbroker.StartWithEndpoint(directory, c.endpoint())
}

func (c *pairedCompanions) endpoint() natsbroker.Endpoint {
	return natsbroker.Endpoint{
		Authentication: companionaccess.New(
			c.devices,
			c.offers,
			wallclock.Clock{},
			pageServicesPermissions(),
			pairingWait,
		),
		Address:     c.address,
		Certificate: c.certificate,
	}
}

func pageServicesPermissions() natspermissions.Permissions {
	return natspermissions.JoinedFrom(
		pagescrapeassembly.ServiceConfig{
			ScrapeRequestDurable: pagescrapeassembly.DefaultScrapeRequestDurable,
		}.BrokerPermissions(),
		pagerwiassembly.ServiceConfig{
			PageOfferDurable: pagerwiassembly.DefaultPageOfferDurable,
		}.BrokerPermissions(),
	)
}

func (c *pairedCompanions) close(ctx context.Context) {
	if err := c.devices.Close(); err != nil {
		slog.WarnContext(ctx, "paired devices did not close", slog.Any("error", err))
	}
}
