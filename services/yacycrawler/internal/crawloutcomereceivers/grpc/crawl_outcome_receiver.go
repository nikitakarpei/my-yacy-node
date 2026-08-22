// Package grpc serves the crawl outcomes contract over gRPC.
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	crawlerv1 "github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawler/v1"
)

const (
	msgReceiverListening = "crawl outcome receiver listening"
	msgReceiverStopped   = "crawl outcome receiver stopped"
)

type CrawlOutcomeReceiver struct {
	crawlOutcomesServer *crawlOutcomesServer
	listenAddress       string
}

func NewCrawlOutcomeReceiver(
	redirectResolutions RedirectResolutions,
	disposedPages DisposedPages,
	listenAddress string,
) *CrawlOutcomeReceiver {
	return &CrawlOutcomeReceiver{
		crawlOutcomesServer: &crawlOutcomesServer{
			redirectResolutions: redirectResolutions,
			disposedPages:       disposedPages,
		},
		listenAddress: listenAddress,
	}
}

func (r *CrawlOutcomeReceiver) Serve(ctx context.Context) error {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", r.listenAddress)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.listenAddress, err)
	}
	endpoint := grpc.NewServer()
	crawlerv1.RegisterCrawlOutcomesServer(endpoint, r.crawlOutcomesServer)
	go stopWhenDone(ctx, endpoint)
	slog.DebugContext(ctx, msgReceiverListening, slog.String("listen", r.listenAddress))
	if err := endpoint.Serve(listener); err != nil {
		return fmt.Errorf("serve crawl outcomes contract: %w", err)
	}
	slog.DebugContext(ctx, msgReceiverStopped)
	return nil
}

func stopWhenDone(ctx context.Context, endpoint *grpc.Server) {
	<-ctx.Done()
	endpoint.GracefulStop()
}
