// Package nodeconfiguration reads every operator setting the node runs on from
// the environment. Each feature owns a settings type and the loader that reads
// it, so a new setting joins the feature it belongs to and nothing else.
package nodeconfiguration

type Settings struct {
	Identity     IdentityConfig
	Serving      ServingConfig
	Storage      StorageConfig
	Escrow       EscrowConfig
	PeerExchange PeerExchangeConfig
	Distribution DistributionConfig
	Crawl        CrawlConfig
	Egress       EgressConfig
}

func Load(getenv func(string) string) (Settings, error) {
	serving, err := loadServingConfig(getenv)
	if err != nil {
		return Settings{}, err
	}

	peerExchange, err := loadPeerExchangeConfig(getenv)
	if err != nil {
		return Settings{}, err
	}

	identity, err := loadIdentityConfig(getenv, serving.PeerAddr, peerExchange.Announcing())
	if err != nil {
		return Settings{}, err
	}

	storage, err := loadStorageConfig(getenv)
	if err != nil {
		return Settings{}, err
	}

	escrow, err := loadEscrowConfig(getenv)
	if err != nil {
		return Settings{}, err
	}

	distribution, err := loadDistributionConfig(getenv)
	if err != nil {
		return Settings{}, err
	}

	egress, err := loadEgressConfig(getenv)
	if err != nil {
		return Settings{}, err
	}

	crawl, err := loadCrawlConfig(getenv)
	if err != nil {
		return Settings{}, err
	}

	return Settings{
		Identity:     identity,
		Serving:      serving,
		Storage:      storage,
		Escrow:       escrow,
		PeerExchange: peerExchange,
		Distribution: distribution,
		Crawl:        crawl,
		Egress:       egress,
	}, nil
}
