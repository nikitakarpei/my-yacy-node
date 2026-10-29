// Command yacy-peer-key prints a new YaCy peer key. It searches the key for the time that
// -search-time sets and keeps the strongest proof of work it finds.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const defaultSearchTime = time.Minute

func main() {
	searchTime := flag.Duration("search-time", defaultSearchTime, "time to search the peer key")
	flag.Parse()
	if err := run(context.Background(), *searchTime, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, searchTime time.Duration, out io.Writer) error {
	slog.InfoContext(ctx, "peer key search started", slog.Duration("searchTime", searchTime))
	key, err := yacymodel.NewPeerKey(ctx, searchTime, wallclock.Clock{})
	if err != nil {
		return fmt.Errorf("generate peer key: %w", err)
	}
	slog.InfoContext(ctx, "peer key made",
		slog.String("peer", key.Hash().String()),
		slog.Int("workBits", key.Work()),
	)
	if err := key.WriteSecret(out); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return fmt.Errorf("write peer key: %w", err)
	}
	return nil
}
