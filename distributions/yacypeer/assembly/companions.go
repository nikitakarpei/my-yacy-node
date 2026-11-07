package yacypeerassembly

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"time"

	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/pairingoffers"
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
	broker, err := natsbroker.StartForCompanions(
		cfg.directoryOf(brokerDirectoryName),
		natsbroker.CompanionListening{
			Authentication: companionaccess.New(
				parts.devices,
				parts.offers,
				wallclock.Clock{},
				companionaccess.DeviceWorkFor(
					pagescrapeassembly.DefaultScrapeRequestDurable,
					pagerwiassembly.DefaultPageOfferDurable,
				),
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

func closeCompanions(ctx context.Context, parts *companions) {
	if parts == nil {
		return
	}
	if err := parts.devices.Close(); err != nil {
		slog.WarnContext(ctx, "paired devices did not close", slog.Any("error", err))
	}
}
