package pagescrapecontract

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type RejectedPage struct {
	PageURL canonicalurl.CanonicalURL `json:"PageURL"`
	Corpus  CorpusName                `json:"Corpus"`
}

func MarshalRejectedPage(rejected RejectedPage) ([]byte, error) {
	data, err := json.Marshal(rejected)
	if err != nil {
		return nil, fmt.Errorf("marshal rejected page: %w", err)
	}
	return data, nil
}

func UnmarshalRejectedPage(data []byte) (RejectedPage, error) {
	var rejected RejectedPage
	if err := json.Unmarshal(data, &rejected); err != nil {
		return RejectedPage{}, fmt.Errorf("unmarshal rejected page: %w", err)
	}
	if rejected.Corpus == "" {
		return RejectedPage{}, fmt.Errorf("unmarshal rejected page: no corpus")
	}
	return rejected, nil
}
