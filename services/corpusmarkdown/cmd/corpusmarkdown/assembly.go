package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownintake"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownstoremetrics"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
)

type objectStoreMarkdown struct {
	store jetstream.ObjectStore
}

func (o objectStoreMarkdown) Put(ctx context.Context, name string, markdown []byte) error {
	if _, err := o.store.PutBytes(ctx, name, markdown); err != nil {
		return fmt.Errorf("put page markdown: %w", err)
	}
	return nil
}

func RunService(ctx context.Context, cfg ServiceConfig) error {
	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		return fmt.Errorf("connect nats: %w", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("init jetstream: %w", err)
	}
	stream, err := js.Stream(
		ctx,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationKindMarkdown),
	)
	if err != nil {
		return fmt.Errorf("lookup crawled page stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.CrawledPageDurable,
		FilterSubject: cfg.CrawledPageSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: cfg.Concurrency,
	})
	if err != nil {
		return fmt.Errorf("create crawled page consumer: %w", err)
	}

	store, err := pagemarkdownstore.EnsureBucket(ctx, js)
	if err != nil {
		return fmt.Errorf("ensure page markdown bucket: %w", err)
	}

	metrics := markdownstoremetrics.New()
	intake := markdownintake.NewPageMarkdownConsumer(
		consumer,
		objectStoreMarkdown{store: store},
		metrics,
		cfg.Concurrency,
	)

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(metrics.Handler()),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, "corpusmarkdown started",
		slog.String("subject", cfg.CrawledPageSubject),
		slog.String("bucket", pagemarkdownstore.BucketName),
		slog.Int("concurrency", cfg.Concurrency),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		func(runCtx context.Context) error {
			if err := intake.Run(runCtx); err != nil {
				return fmt.Errorf("run page markdown consumer: %w", err)
			}
			return nil
		},
	)
	slog.InfoContext(ctx, "corpusmarkdown stopped")
	return err
}
