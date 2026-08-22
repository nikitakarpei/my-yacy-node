package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	crawlorderplacersjetstream "github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/internal/crawlorderplacers/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/internal/firecrawlscrape"
	"github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/internal/markdownrecall"
	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	crawlerv1 "github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawler/v1"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	scrapeServerName  = "scrape"
	msgServiceStarted = "firecrawlshim started"
	msgServiceStopped = "firecrawlshim stopped"
)

func RunService(ctx context.Context, cfg ServiceConfig) error {
	crawlJetStream, crawlConnection, err := jetstreamconnect.Open(cfg.CrawlNATSURL)
	if err != nil {
		return err
	}
	defer func() { _ = crawlConnection.Close() }()

	crawlOutcomes, closeCrawlOutcomes, err := dialCrawlOutcomes(cfg.CrawlOutcomesTarget)
	if err != nil {
		return err
	}
	defer closeCrawlOutcomes()
	markdownCorpus, closeMarkdownCorpus, err := dialMarkdownCorpus(cfg.MarkdownCorpusTarget)
	if err != nil {
		return err
	}
	defer closeMarkdownCorpus()

	recaller := markdownrecall.NewMarkdownRecaller(
		crawlOutcomes,
		crawlorderplacersjetstream.NewCrawlOrderPlacement(crawlJetStream, cfg.OrdersSubject),
		markdownCorpus,
		recallConfigFrom(cfg),
	)

	announceServiceStarted(ctx, cfg)
	err = servergroup.Run(ctx, shutdownLimit, scrapeServersFor(cfg, recaller))
	announceServiceStopped(ctx)
	return err
}

func dialCrawlOutcomes(target string) (crawlerv1.CrawlOutcomesClient, func(), error) {
	connection, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("dial crawl outcomes %s: %w", target, err)
	}
	return crawlerv1.NewCrawlOutcomesClient(connection),
		func() { _ = connection.Close() },
		nil
}

func dialMarkdownCorpus(target string) (corpusmarkdownv1.MarkdownCorpusClient, func(), error) {
	connection, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("dial markdown corpus %s: %w", target, err)
	}
	return corpusmarkdownv1.NewMarkdownCorpusClient(connection),
		func() { _ = connection.Close() },
		nil
}

func recallConfigFrom(cfg ServiceConfig) markdownrecall.Config {
	return markdownrecall.Config{
		RecallLimit:        cfg.RecallLimit,
		PollInterval:       cfg.PollInterval,
		MaxRecallsInFlight: cfg.MaxInFlight,
	}
}

func announceServiceStarted(ctx context.Context, cfg ServiceConfig) {
	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listen", cfg.ListenAddr),
		slog.String("crawlOutcomesTarget", cfg.CrawlOutcomesTarget),
		slog.String("markdownCorpusTarget", cfg.MarkdownCorpusTarget),
		slog.Duration("recallLimit", cfg.RecallLimit),
	)
}

func scrapeServersFor(
	cfg ServiceConfig,
	recaller *markdownrecall.MarkdownRecaller,
) []servergroup.NamedServer {
	mux := http.NewServeMux()
	mux.Handle("POST /v1/scrape", firecrawlscrape.NewScraper(recaller))
	return []servergroup.NamedServer{{
		Name: scrapeServerName,
		Server: &http.Server{
			Addr:              cfg.ListenAddr,
			Handler:           mux,
			ReadHeaderTimeout: readHeaderLimit,
		},
	}}
}

func announceServiceStopped(ctx context.Context) {
	slog.InfoContext(ctx, msgServiceStopped)
}
