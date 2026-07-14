package pagepublication

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerwi"
)

type PageRWIOutput struct {
	publisher jetstream.JetStream
	subject   string
	text      crawlcapability.ContentDerivation
}

func NewPageRWIOutput(
	publisher jetstream.JetStream,
	subject string,
	text crawlcapability.ContentDerivation,
) PageRWIOutput {
	return PageRWIOutput{publisher: publisher, subject: subject, text: text}
}

func (PageRWIOutput) Name() string {
	return string(yacycrawlcontract.PageRepresentationRWI)
}

func (o PageRWIOutput) Accepts(format crawlcapability.PageContentFormat) bool {
	return acceptsSourceFormat(o.text, format)
}

func (o PageRWIOutput) Publish(ctx context.Context, page crawlcapability.CrawledPage) error {
	representation, err := pagerwi.Build(page, o.text)
	if err != nil {
		return fmt.Errorf("build page rwi representation: %w", err)
	}
	for _, chunk := range chunkPageRWI(representation) {
		payload, err := yacycrawlcontract.MarshalPageRWIChunk(chunk)
		if err != nil {
			return fmt.Errorf("marshal page rwi chunk: %w", err)
		}
		if _, err := o.publisher.Publish(ctx, o.subject, payload); err != nil {
			return classifyPublishError(err)
		}
	}
	return nil
}
