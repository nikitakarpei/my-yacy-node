package pagerwicontract

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageRWI struct {
	PageURL     canonicalurl.CanonicalURL `json:"PageURL"`
	Metadata    yacymodel.URLMetadata     `json:"Metadata"`
	Postings    []yacymodel.RWIPosting    `json:"Postings,omitempty"`
	Attestation attestedstatements.Bundle `json:"Attestation,omitzero"`
}

func (r PageRWI) AttestedBy(bundle attestedstatements.Bundle) PageRWI {
	r.Attestation = bundle

	return r
}

func MarshalPageRWI(pageRWI PageRWI) ([]byte, error) {
	data, err := json.Marshal(pageRWI)
	if err != nil {
		return nil, fmt.Errorf("marshal rwi of page: %w", err)
	}
	return data, nil
}

func UnmarshalPageRWI(data []byte) (PageRWI, error) {
	var pageRWI PageRWI
	if err := json.Unmarshal(data, &pageRWI); err != nil {
		return PageRWI{}, fmt.Errorf("unmarshal rwi of page: %w", err)
	}
	return pageRWI, nil
}
