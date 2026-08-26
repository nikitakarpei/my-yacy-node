// Package text writes the scrape requests the command selects as one url per line, so
// that an operator can read what a run would ask for before it asks for it.
package text

import (
	"context"
	"fmt"
	"io"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Publisher struct {
	requests io.Writer
}

func New(requests io.Writer) *Publisher {
	return &Publisher{requests: requests}
}

func (p *Publisher) Publish(_ context.Context, canonicalURL canonicalurl.CanonicalURL) error {
	if _, err := fmt.Fprintln(p.requests, canonicalURL.String()); err != nil {
		return fmt.Errorf("write scrape request %s: %w", canonicalURL, err)
	}
	return nil
}

func (p *Publisher) Close() {}
