package pagefeed

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type crawledPageSubject struct {
	stream  jetstream.JetStream
	subject string
}

func (s crawledPageSubject) Publish(
	ctx context.Context,
	publication crawlcapability.PagePublication,
) error {
	for _, message := range publication.Messages() {
		if _, err := s.stream.Publish(ctx, s.subject, message); err != nil {
			return classifyPublishError(err)
		}
	}
	return nil
}
