// Command yacy-peer-key prints a new YaCy peer key.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(out io.Writer) error {
	key, err := yacymodel.NewPeerKey(context.Background())
	if err != nil {
		return fmt.Errorf("generate peer key: %w", err)
	}
	if _, err := fmt.Fprintln(out, key); err != nil {
		return fmt.Errorf("write peer key: %w", err)
	}
	return nil
}
