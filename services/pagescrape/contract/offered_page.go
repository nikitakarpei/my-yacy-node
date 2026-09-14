package pagescrapecontract

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type OfferedPage struct {
	PageURL          canonicalurl.CanonicalURL `json:"PageURL"`
	LandedURL        canonicalurl.CanonicalURL `json:"LandedURL"`
	ContentType      string                    `json:"ContentType"`
	Body             []byte                    `json:"Body"`
	RobotsDirectives []string                  `json:"RobotsDirectives,omitzero"`
}

func OfferedPageFrom(
	request ScrapeRequest,
	fetchedPage pagefetch.FetchedPage,
	landedURL canonicalurl.CanonicalURL,
) OfferedPage {
	return OfferedPage{
		PageURL:          request.PageURL,
		LandedURL:        landedURL,
		ContentType:      fetchedPage.ContentType,
		Body:             fetchedPage.Body,
		RobotsDirectives: fetchedPage.RobotsDirectives,
	}
}

func MarshalOfferedPage(page OfferedPage) ([]byte, error) {
	data, err := json.Marshal(page)
	if err != nil {
		return nil, fmt.Errorf("marshal offered page: %w", err)
	}
	return data, nil
}

func UnmarshalOfferedPage(data []byte) (OfferedPage, error) {
	var page OfferedPage
	if err := json.Unmarshal(data, &page); err != nil {
		return OfferedPage{}, fmt.Errorf("unmarshal offered page: %w", err)
	}
	return page, nil
}
