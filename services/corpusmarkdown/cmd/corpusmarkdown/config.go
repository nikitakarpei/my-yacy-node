package main

import (
	pagemarkdowncorporas3 "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/s3"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvPageOfferNATSURL              = "SCRAPE_PAGE_OFFER_NATS_URL"
	EnvPageMarkdownS3Endpoint        = "PAGE_MARKDOWN_S3_ENDPOINT"
	EnvPageMarkdownS3Region          = "PAGE_MARKDOWN_S3_REGION"
	EnvPageMarkdownS3AccessKeyID     = "PAGE_MARKDOWN_S3_ACCESS_KEY_ID"
	EnvPageMarkdownS3SecretAccessKey = "PAGE_MARKDOWN_S3_SECRET_ACCESS_KEY"
	EnvPageMarkdownBucket            = "PAGE_MARKDOWN_BUCKET"
	EnvPageOfferDurable              = "SCRAPE_PAGE_OFFER_DURABLE"
	EnvPageOfferIntakeConcurrency    = "SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY"
	EnvListenAddr                    = "CORPUSMARKDOWN_LISTEN_ADDR"
	EnvOpsAddr                       = "CORPUSMARKDOWN_OPS_ADDR"

	DefaultListenAddr                 = ":8094"
	DefaultOpsAddr                    = ":9090"
	DefaultPageOfferDurable           = "corpusmarkdown"
	DefaultPageMarkdownS3Region       = "us-east-1"
	DefaultPageMarkdownBucket         = "yacy-page-markdown"
	DefaultPageOfferIntakeConcurrency = 4
)

type ServiceConfig struct {
	PageOfferNATSURL           string
	PageMarkdownObjectStore    pagemarkdowncorporas3.ObjectStore
	PageMarkdownBucket         string
	PageOfferDurable           string
	PageOfferIntakeConcurrency int
	ListenAddr                 string
	OpsAddr                    string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	pageOfferNATSURL, err := envconfig.Required(getenv, EnvPageOfferNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageMarkdownObjectStore, err := loadPageMarkdownObjectStore(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageOfferIntakeConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvPageOfferIntakeConcurrency,
		DefaultPageOfferIntakeConcurrency,
	)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		PageOfferNATSURL:        pageOfferNATSURL,
		PageMarkdownObjectStore: pageMarkdownObjectStore,
		PageMarkdownBucket: envconfig.String(
			getenv,
			EnvPageMarkdownBucket,
			DefaultPageMarkdownBucket,
		),
		PageOfferDurable: envconfig.String(
			getenv,
			EnvPageOfferDurable,
			DefaultPageOfferDurable,
		),
		PageOfferIntakeConcurrency: pageOfferIntakeConcurrency,
		ListenAddr:                 envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:                    envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}

func loadPageMarkdownObjectStore(
	getenv func(string) string,
) (pagemarkdowncorporas3.ObjectStore, error) {
	endpoint, err := envconfig.RequiredHTTPURL(getenv, EnvPageMarkdownS3Endpoint)
	if err != nil {
		return pagemarkdowncorporas3.ObjectStore{}, err
	}
	accessKeyID, err := envconfig.Required(getenv, EnvPageMarkdownS3AccessKeyID)
	if err != nil {
		return pagemarkdowncorporas3.ObjectStore{}, err
	}
	secretAccessKey, err := envconfig.Required(getenv, EnvPageMarkdownS3SecretAccessKey)
	if err != nil {
		return pagemarkdowncorporas3.ObjectStore{}, err
	}
	return pagemarkdowncorporas3.ObjectStore{
		Endpoint: endpoint.String(),
		Region: envconfig.String(
			getenv,
			EnvPageMarkdownS3Region,
			DefaultPageMarkdownS3Region,
		),
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}, nil
}
