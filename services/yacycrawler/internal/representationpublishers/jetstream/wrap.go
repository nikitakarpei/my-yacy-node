package jetstream

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
)

// Framer is the pure part of a page representation: it turns a page's content
// into messages, without knowing how those messages are transported.
type Framer interface {
	Kind() yacycrawlcontract.PageRepresentationKind
	ContentFormat() contentformatgraph.Format
	Frame(page pagepublication.Page, content []byte) ([][]byte, error)
}

type representation struct {
	Framer
	stream  jetstream.JetStream
	subject string
}

// Wrap adds JetStream publication to a pure Framer, addressed to subject.
func Wrap(
	pure Framer,
	subject string,
	stream jetstream.JetStream,
) pagepublication.PageRepresentation {
	return &representation{Framer: pure, stream: stream, subject: subject}
}

func (r *representation) Publish(
	ctx context.Context,
	messages [][]byte,
) error {
	for _, message := range messages {
		if _, err := r.stream.Publish(ctx, r.subject, message); err != nil {
			return classifyPublishError(err)
		}
	}
	return nil
}
