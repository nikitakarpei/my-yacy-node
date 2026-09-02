package pagescrapecontract

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type KeptPage struct {
	PageURL canonicalurl.CanonicalURL `json:"PageURL"`
	Corpus  CorpusName                `json:"Corpus"`
}

func MarshalKeptPage(kept KeptPage) ([]byte, error) {
	data, err := json.Marshal(kept)
	if err != nil {
		return nil, fmt.Errorf("marshal kept page: %w", err)
	}
	return data, nil
}

func UnmarshalKeptPage(data []byte) (KeptPage, error) {
	var kept KeptPage
	if err := json.Unmarshal(data, &kept); err != nil {
		return KeptPage{}, fmt.Errorf("unmarshal kept page: %w", err)
	}
	if kept.Corpus == "" {
		return KeptPage{}, fmt.Errorf("unmarshal kept page: no corpus")
	}
	return kept, nil
}
