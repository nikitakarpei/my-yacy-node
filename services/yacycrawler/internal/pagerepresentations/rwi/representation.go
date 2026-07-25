package rwi

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
)

const postingsPerChunkLimit = 1000

type Representation struct{}

func New() Representation {
	return Representation{}
}

func (Representation) Kind() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindRWI
}

func (Representation) ContentFormat() contentformatgraph.Format {
	return contentformatgraph.FormatFullText
}

func (Representation) Frame(
	page pagepublication.Page,
	content []byte,
) ([][]byte, error) {
	representation, err := buildRepresentation(page, content)
	if err != nil {
		return nil, err
	}
	messages := make([][]byte, 0, len(representation.Postings)/postingsPerChunkLimit+1)
	for _, chunk := range chunkPageRWI(representation) {
		message, err := yacycrawlcontract.MarshalPageRWIChunk(chunk)
		if err != nil {
			return nil, fmt.Errorf(
				"marshal page rwi chunk: %w", err,
			)
		}
		messages = append(messages, message)
	}
	return messages, nil
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
