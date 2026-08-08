// Package grpc serves the recall contract over gRPC.
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall/pagerecall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

const (
	msgReceiverListening = "recall receiver listening"
	msgReceiverStopped   = "recall receiver stopped"
)

type RecallReceiver struct {
	recallServer  *recallServer
	listenAddress string
}

func NewRecallReceiver(
	recaller Recaller,
	corpora []pagerecall.Corpus,
	listenAddress string,
) (*RecallReceiver, error) {
	recallServer, err := newRecallServer(recaller, corpora)
	if err != nil {
		return nil, err
	}
	return &RecallReceiver{recallServer: recallServer, listenAddress: listenAddress}, nil
}

func (r *RecallReceiver) Serve(ctx context.Context) error {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", r.listenAddress)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.listenAddress, err)
	}
	endpoint := grpc.NewServer()
	corpusrecallv1.RegisterRecallServer(endpoint, r.recallServer)
	go stopWhenDone(ctx, endpoint)
	slog.DebugContext(ctx, msgReceiverListening, slog.String("listen", r.listenAddress))
	if err := endpoint.Serve(listener); err != nil {
		return fmt.Errorf("serve recall contract: %w", err)
	}
	slog.DebugContext(ctx, msgReceiverStopped)
	return nil
}

func stopWhenDone(ctx context.Context, endpoint *grpc.Server) {
	<-ctx.Done()
	endpoint.GracefulStop()
}
