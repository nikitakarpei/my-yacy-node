package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/crawlrequest"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/markdownrecall"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerecall"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recallgrpc"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recallmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/redirectlookup"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	opsShutdownLimit   = 15 * time.Second
)

func RunService(ctx context.Context, cfg ServiceConfig) error {
	js, conn, err := jetstreamconnect.Open(cfg.NATSURL)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	if err := yacycrawlcontract.EnsureOrdersStream(ctx, js, yacycrawlcontract.OrdersStreamSpec{
		Subject: cfg.OrdersSubject,
	}); err != nil {
		return fmt.Errorf("ensure orders stream: %w", err)
	}

	markdownStore, err := pagemarkdownstore.EnsureBucket(ctx, js)
	if err != nil {
		return fmt.Errorf("open page markdown bucket: %w", err)
	}
	redirects, err := js.KeyValue(ctx, yacycrawlcontract.RedirectResolutionBucketName)
	if err != nil {
		return fmt.Errorf("open redirect resolution bucket: %w", err)
	}

	metrics := recallmetrics.New()
	placer := crawlrequest.NewOrderPlacement(js, cfg.OrdersSubject)
	resolver := redirectlookup.NewReader(redirects)
	markdownSource := markdownrecall.NewSource(markdownStore, cfg.MaxResponseBytes)
	kinds := []representationKind{
		markdownRepresentation(markdownSource),
	}
	recaller := pagerecall.NewRecaller(
		placer,
		resolver,
		recallSources(kinds),
		metrics,
		pagerecall.Config{
			Deadline:     cfg.Deadline,
			PollInterval: cfg.PollInterval,
			MaxInFlight:  cfg.MaxInFlight,
		},
	)

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.ListenAddr, err)
	}
	grpcServer := grpc.NewServer()
	recallServer := recallgrpc.NewRecallServer(recaller, representationCodecs(kinds))
	corpusrecallv1.RegisterRecallServer(grpcServer, recallServer)

	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(metrics.Handler()),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, "corpusrecall started",
		slog.String("listen", cfg.ListenAddr),
		slog.String("ordersSubject", cfg.OrdersSubject),
		slog.Duration("deadline", cfg.Deadline),
	)
	err = servergroup.Run(ctx, opsShutdownLimit,
		[]servergroup.NamedServer{{Name: "ops", Server: opsServer}},
		func(runCtx context.Context) error {
			go func() {
				<-runCtx.Done()
				grpcServer.GracefulStop()
			}()
			if err := grpcServer.Serve(listener); err != nil {
				return fmt.Errorf("serve grpc: %w", err)
			}
			return nil
		},
	)
	slog.InfoContext(ctx, "corpusrecall stopped")
	return err
}
