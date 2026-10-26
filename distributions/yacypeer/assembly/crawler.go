package yacypeerassembly

import (
	"context"

	yacycrawlerassembly "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
)

type crawler struct {
	service service
	apiURL  string
}

func crawlerFor(
	ctx context.Context,
	cfg PeerConfig,
	addresses peerAddresses,
	getenv func(string) string,
) (crawler, error) {
	assembled, err := assembledServiceOf(
		ctx,
		yacycrawlerService,
		serviceenvironment.Wiring{
			yacycrawlerassembly.EnvAPIAddr:       loopback.AnyFreeAddress,
			yacycrawlerassembly.EnvCrawlNATSURL:  addresses.natsURL,
			yacycrawlerassembly.EnvCrawlStateDir: cfg.directoryOf(yacycrawlerService),
			yacycrawlerassembly.EnvProxyURL:      addresses.egressProxyURL.String(),
			yacycrawlerassembly.EnvProxyDialMode: yacycrawlerassembly.DefaultProxyDialMode,
		},
		yacycrawlerassembly.LoadServiceConfig,
		yacycrawlerassembly.ListenersFor,
		yacycrawlerassembly.RunService,
		getenv,
	)
	if err != nil {
		return crawler{}, err
	}
	return crawler{service: assembled.service(), apiURL: assembled.listeners.APIURL()}, nil
}
