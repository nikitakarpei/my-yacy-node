package pagepublication

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const postingsPerChunkLimit = 1000

type RWIPublication struct {
	publisher jetstream.JetStream
	subject   string
}

func NewRWIPublication(publisher jetstream.JetStream, subject string) RWIPublication {
	return RWIPublication{publisher: publisher, subject: subject}
}

func (p RWIPublication) Publish(
	ctx context.Context,
	representation yacycrawlcontract.PageRWIRepresentation,
) error {
	for _, chunk := range chunkPageRWI(representation) {
		payload, err := yacycrawlcontract.MarshalPageRWIChunk(chunk)
		if err != nil {
			return fmt.Errorf("marshal page rwi chunk: %w", err)
		}
		if _, err := p.publisher.Publish(ctx, p.subject, payload); err != nil {
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
