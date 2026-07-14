package yacycrawlcontract

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	chunkKindMetadata = "metadata"
	chunkKindPosting  = "posting"
)

type PageRWIChunk interface {
	envelope() pageRWIChunkEnvelope
}

type PageRWIMetadataChunk struct {
	CanonicalURL string
	Metadata     []yacymodel.URIMetadataRow
}

type PageRWIPostingChunk struct {
	CanonicalURL string
	Postings     []yacymodel.RWIPosting
}

type pageRWIChunkEnvelope struct {
	Kind         string
	CanonicalURL string
	Metadata     []yacymodel.URIMetadataRow `json:",omitempty"`
	Postings     []yacymodel.RWIPosting     `json:",omitempty"`
}

func (c PageRWIMetadataChunk) envelope() pageRWIChunkEnvelope {
	return pageRWIChunkEnvelope{
		Kind:         chunkKindMetadata,
		CanonicalURL: c.CanonicalURL,
		Metadata:     c.Metadata,
	}
}

func (c PageRWIPostingChunk) envelope() pageRWIChunkEnvelope {
	return pageRWIChunkEnvelope{
		Kind:         chunkKindPosting,
		CanonicalURL: c.CanonicalURL,
		Postings:     c.Postings,
	}
}

func MarshalPageRWIChunk(chunk PageRWIChunk) ([]byte, error) {
	data, err := json.Marshal(chunk.envelope())
	if err != nil {
		return nil, fmt.Errorf("marshal page rwi chunk: %w", err)
	}
	return data, nil
}

func UnmarshalPageRWIChunk(data []byte) (PageRWIChunk, error) {
	var envelope pageRWIChunkEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal page rwi chunk: %w", err)
	}
	switch envelope.Kind {
	case chunkKindMetadata:
		return PageRWIMetadataChunk{
			CanonicalURL: envelope.CanonicalURL,
			Metadata:     envelope.Metadata,
		}, nil
	case chunkKindPosting:
		return PageRWIPostingChunk{
			CanonicalURL: envelope.CanonicalURL,
			Postings:     envelope.Postings,
		}, nil
	default:
		return nil, fmt.Errorf("unmarshal page rwi chunk: unknown kind %q", envelope.Kind)
	}
}
