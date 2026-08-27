// Package noop accepts every scrape request and sends none, so a run selects and lists pages
// without a broker and without work for the crawl.
package noop

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Publisher struct{}

func Open() Publisher {
	return Publisher{}
}

func (Publisher) Publish(
	context.Context,
	canonicalurl.CanonicalURL,
	canonicalurl.CanonicalURL,
) error {
	return nil
}

func (Publisher) Close() {}
