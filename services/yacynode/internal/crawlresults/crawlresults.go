// Package crawlresults absorbs the RWI chunks returned by the crawl fleet. Each
// chunk carries either a page's URL metadata or one bounded batch of its postings,
// stored through the node's existing receivers. NewIngestConsumer and its Run loop
// are the only surface; IngestStream is the port chunks arrive through.
package crawlresults

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type IngestDelivery struct {
	Chunk yacycrawlcontract.PageRWIChunk
	Ack   func(context.Context) error
	Nak   func(context.Context) error
	Term  func(context.Context) error
}

type IngestStream interface {
	Receive() <-chan IngestDelivery
}

type IngestConsumer struct {
	stream   IngestStream
	urls     urlmeta.URLReceiver
	postings rwipostings.PostingReceiver
}

func NewIngestConsumer(
	stream IngestStream,
	urls urlmeta.URLReceiver,
	postings rwipostings.PostingReceiver,
) *IngestConsumer {
	return &IngestConsumer{stream: stream, urls: urls, postings: postings}
}
