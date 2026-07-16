package pagefeed

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerwi"
)

const postingsPerChunkLimit = 1000

type RWIFeed struct {
	publisher     jetstream.JetStream
	subject       string
	contentFormat crawlcapability.PageContentFormat
}

func NewRWIFeed(
	publisher jetstream.JetStream,
	subject string,
	contentFormat crawlcapability.PageContentFormat,
) RWIFeed {
	return RWIFeed{publisher: publisher, subject: subject, contentFormat: contentFormat}
}

func (RWIFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindRWI
}

func (f RWIFeed) ContentFormat() crawlcapability.PageContentFormat {
	return f.contentFormat
}

func (f RWIFeed) Derive(
	page crawlcapability.CrawledPage,
	content []byte,
) (crawlcapability.PagePublication, error) {
	representation, err := pagerwi.Build(page, content)
	if err != nil {
		return crawlcapability.PagePublication{}, err
	}
	messages := make([][]byte, 0, len(representation.Postings)/postingsPerChunkLimit+1)
	for _, chunk := range chunkPageRWI(representation) {
		message, err := yacycrawlcontract.MarshalPageRWIChunk(chunk)
		if err != nil {
			return crawlcapability.PagePublication{}, fmt.Errorf(
				"marshal page rwi chunk: %w", err,
			)
		}
		messages = append(messages, message)
	}
	return crawlcapability.NewPagePublication(messages...), nil
}

func (f RWIFeed) Publish(ctx context.Context, publication crawlcapability.PagePublication) error {
	for _, message := range publication.Messages() {
		if _, err := f.publisher.Publish(ctx, f.subject, message); err != nil {
			return classifyPublishError(err)
		}
	}
	return nil
}

func chunkPageRWI(
	representation yacycrawlcontract.PageRWIRepresentation,
) []yacycrawlcontract.PageRWIChunk {
	chunks := []yacycrawlcontract.PageRWIChunk{
		yacycrawlcontract.PageRWIMetadataChunk{
			CanonicalURL: representation.CanonicalURL,
			Metadata:     representation.Metadata,
		},
	}
	for start := 0; start < len(representation.Postings); start += postingsPerChunkLimit {
		end := min(start+postingsPerChunkLimit, len(representation.Postings))
		chunks = append(chunks, yacycrawlcontract.PageRWIPostingChunk{
			CanonicalURL: representation.CanonicalURL,
			Postings:     representation.Postings[start:end],
		})
	}
	return chunks
}
