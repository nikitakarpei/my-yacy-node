// Package grpc serves the markdown corpus contract over gRPC.
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
)

const (
	msgReceiverListening = "markdown recall receiver listening"
	msgReceiverStopped   = "markdown recall receiver stopped"
)

type MarkdownRecallReceiver struct {
	markdownCorpusServer *markdownCorpusServer
	listenAddress        string
}

func NewMarkdownRecallReceiver(
	recall PageMarkdownRecall,
	listenAddress string,
) *MarkdownRecallReceiver {
	return &MarkdownRecallReceiver{
		markdownCorpusServer: &markdownCorpusServer{recall: recall},
		listenAddress:        listenAddress,
	}
}

func (r *MarkdownRecallReceiver) Serve(ctx context.Context) error {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", r.listenAddress)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.listenAddress, err)
	}
	endpoint := grpc.NewServer()
	corpusmarkdownv1.RegisterMarkdownCorpusServer(endpoint, r.markdownCorpusServer)
	go stopWhenDone(ctx, endpoint)
	slog.DebugContext(ctx, msgReceiverListening, slog.String("listen", r.listenAddress))
	if err := endpoint.Serve(listener); err != nil {
		return fmt.Errorf("serve markdown corpus contract: %w", err)
	}
	slog.DebugContext(ctx, msgReceiverStopped)
	return nil
}

func stopWhenDone(ctx context.Context, endpoint *grpc.Server) {
	<-ctx.Done()
	endpoint.GracefulStop()
}
