package pagefeed

import (
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerwi"
)

const postingsPerChunkLimit = 1000

type RWIFeed struct {
	crawledPageSubject
}

func NewRWIFeed(stream jetstream.JetStream, subject string) RWIFeed {
	return RWIFeed{crawledPageSubject{stream: stream, subject: subject}}
}

func (RWIFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindRWI
}

func (RWIFeed) ContentFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatText
}

func (RWIFeed) Derive(
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
