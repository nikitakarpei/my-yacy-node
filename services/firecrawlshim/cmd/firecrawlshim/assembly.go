package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
	"github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/internal/firecrawlscrape"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	readHeaderLimit = 10 * time.Second
	shutdownLimit   = 15 * time.Second
)

func RunService(ctx context.Context, cfg ServiceConfig) error {
	conn, err := grpc.NewClient(
		cfg.RecallTarget,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("dial corpusrecall %s: %w", cfg.RecallTarget, err)
	}
	defer func() { _ = conn.Close() }()

	scraper := firecrawlscrape.NewScraper(corpusrecallv1.NewRecallClient(conn), cfg.RecallTimeout)
	mux := http.NewServeMux()
	mux.Handle("POST /v1/scrape", scraper)

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, "firecrawlshim started",
		slog.String("listen", cfg.ListenAddr),
		slog.String("recallTarget", cfg.RecallTarget),
	)
	err = servergroup.Run(ctx, shutdownLimit,
		[]servergroup.NamedServer{{Name: "scrape", Server: server}},
	)
	slog.InfoContext(ctx, "firecrawlshim stopped")
	return err
}
