package pagepublication

import (
	"context"
	"fmt"
	"strconv"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	postingsPerChunkLimit = 1000
	documentTypeText      = 't'
)

type RWIPublication struct {
	publisher jetstream.JetStream
	subject   string
}

func NewRWIPublication(publisher jetstream.JetStream, subject string) RWIPublication {
	return RWIPublication{publisher: publisher, subject: subject}
}

func (p RWIPublication) Publish(
	ctx context.Context,
	representation crawlcapability.RWIRepresentation,
) error {
	metadata, err := metadataRowOf(representation)
	if err != nil {
		return fmt.Errorf("build page rwi metadata: %w", err)
	}
	for _, chunk := range chunkPageRWI(representation, metadata) {
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

func metadataRowOf(
	representation crawlcapability.RWIRepresentation,
) (yacymodel.URIMetadataRow, error) {
	urlHash, err := yacymodel.HashURL(representation.CanonicalURL)
	if err != nil {
		return yacymodel.URIMetadataRow{}, fmt.Errorf("hash url: %w", err)
	}
	return yacymodel.URIMetadataRow{Properties: map[string]string{
		yacymodel.URLMetaHash: urlHash.String(),
		"dt":                  string(rune(documentTypeText)),
		"url": yacymodel.EncodeBase64WireForm(
			representation.CanonicalURL,
		),
		yacymodel.URLMetaColDescription: yacymodel.EncodeBase64WireForm(representation.Title),
		"size":                          strconv.Itoa(representation.TextLength),
		"wc":                            strconv.Itoa(representation.WordCount),
		"llocal":                        strconv.Itoa(representation.LocalLinkCount),
		"lother":                        strconv.Itoa(representation.ExternalLinkCount),
	}}, nil
}

func chunkPageRWI(
	representation crawlcapability.RWIRepresentation,
	metadata yacymodel.URIMetadataRow,
) []yacycrawlcontract.PageRWIChunk {
	chunks := []yacycrawlcontract.PageRWIChunk{
		yacycrawlcontract.PageRWIMetadataChunk{
			CanonicalURL: representation.CanonicalURL,
			Metadata:     []yacymodel.URIMetadataRow{metadata},
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
