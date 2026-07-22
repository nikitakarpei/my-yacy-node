package jetstream

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
)

// Framer is the pure part of a page feed: it turns a crawled page's content
// into a publication, without knowing how that publication is transported.
type Framer interface {
	Representation() yacycrawlcontract.PageRepresentationKind
	ContentFormat() contentformatgraph.Format
	Frame(page pageabsorption.CrawledPage, content []byte) (pageabsorption.PagePublication, error)
}

type feed struct {
	Framer
	stream  jetstream.JetStream
	subject string
}

// Wrap adds JetStream publication to a pure feed, addressed to subject.
func Wrap(pure Framer, subject string, stream jetstream.JetStream) pageabsorption.Feed {
	return &feed{Framer: pure, stream: stream, subject: subject}
}

func (f *feed) Publish(
	ctx context.Context,
	publication pageabsorption.PagePublication,
) error {
	for _, message := range publication.Messages() {
		if _, err := f.stream.Publish(ctx, f.subject, message); err != nil {
			return classifyPublishError(err)
		}
	}
	return nil
}
