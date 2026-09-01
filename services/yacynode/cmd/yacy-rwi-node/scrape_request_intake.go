package main

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	scrapeprogressobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scrapeprogressobservers/applog"
	scrapeprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scrapeprogressobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scraperequestbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scraperequestintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type scrapeRequestIntake struct {
	broker   *scraperequestbroker.ScrapeRequestBroker
	consumer *scraperequestintake.ScrapeRequestConsumer
}

func openScrapeRequestIntake(
	ctx context.Context,
	config nodeconfiguration.ScrapeRequestIntakeConfig,
	urls urlmeta.URLReceiver,
	postings rwiadmission.PostingReceiver,
	registry prometheus.Registerer,
) (*scrapeRequestIntake, error) {
	broker, err := scraperequestbroker.Open(ctx, scraperequestbroker.Config{
		ScrapeRequestNATSURL:           config.ScrapeRequestNATSURL,
		ScrapeRequestSubject:           config.ScrapeRequestSubject,
		ScrapeRequestDurable:           config.ScrapeRequestDurable,
		ScrapeRequestIntakeConcurrency: config.ScrapeRequestIntakeConcurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("open scrape request broker: %w", err)
	}
	formatDerivations, err := pageformats.New()
	if err != nil {
		broker.Close()

		return nil, err
	}

	return &scrapeRequestIntake{
		broker: broker,
		consumer: scraperequestintake.NewScrapeRequestConsumer(
			scraperequestintake.ScrapeRequestConsumerConfig{
				ScrapeRequestSource: broker.ScrapeRequests,
				PageFetcher: pagefetchershttp.New(
					config.ProxyURL,
					config.ProxyDialMode,
					config.UserAgent,
					config.MaxBodyBytes,
					config.FetchDeadline,
				),
				FormatDerivations: formatDerivations,
				URLReceiver:       urls,
				PostingReceiver:   postings,
				ScrapeProgress: scraperequestintake.ScrapeProgressObservers{
					scrapeprogressobserversapplog.ScrapeProgressLog{},
					scrapeprogressobserversprometheus.New(registry),
				},
				ScrapeRequestIntakeConcurrency: config.ScrapeRequestIntakeConcurrency,
			}),
	}, nil
}

func (i *scrapeRequestIntake) Run(ctx context.Context) error {
	return i.consumer.Run(ctx)
}

func (i *scrapeRequestIntake) Close() {
	i.broker.Close()
}
