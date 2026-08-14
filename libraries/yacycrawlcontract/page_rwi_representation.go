package yacycrawlcontract

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageRWIRepresentation struct {
	CanonicalURL string                  `json:"CanonicalURL"`
	Metadata     []yacymodel.URLMetadata `json:"Metadata"`
	Postings     []yacymodel.RWIPosting  `json:"Postings"`
}

func PageRWIRepresentationFromChunks(
	chunks []PageRWIChunk,
) (PageRWIRepresentation, error) {
	var metadataChunk PageRWIMetadataChunk
	var metadataChunkCount int
	for _, chunk := range chunks {
		if chunk, ok := chunk.(PageRWIMetadataChunk); ok {
			metadataChunkCount++
			metadataChunk = chunk
		}
	}
	if metadataChunkCount == 0 {
		return PageRWIRepresentation{}, errors.New(
			"join page rwi chunks: no metadata chunk",
		)
	}
	if metadataChunkCount > 1 {
		return PageRWIRepresentation{}, errors.New(
			"join page rwi chunks: more than one metadata chunk",
		)
	}

	representation := PageRWIRepresentation{
		CanonicalURL: metadataChunk.CanonicalURL,
		Metadata:     metadataChunk.Metadata,
	}
	for _, chunk := range chunks {
		postingChunk, ok := chunk.(PageRWIPostingChunk)
		if !ok {
			continue
		}
		if postingChunk.CanonicalURL != representation.CanonicalURL {
			return PageRWIRepresentation{}, fmt.Errorf(
				"join page rwi chunks: posting chunk canonical url %q disagrees with metadata chunk canonical url %q",
				postingChunk.CanonicalURL,
				representation.CanonicalURL,
			)
		}
		representation.Postings = append(representation.Postings, postingChunk.Postings...)
	}
	return representation, nil
}
