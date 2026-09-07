package yacysearchendpoint

import (
	"strconv"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type searchPage struct {
	Channels []searchChannel `json:"channels"`
}

type searchChannel struct {
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Link         string       `json:"link"`
	TotalResults string       `json:"totalResults"`
	StartIndex   string       `json:"startIndex"`
	ItemsPerPage string       `json:"itemsPerPage"`
	Items        []searchItem `json:"items"`
}

type searchItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description"`
	PubDate     string `json:"pubDate"`
	Image       string `json:"image"`
}

const channelTitle = "YaCy Search Engine: DHT search"

func searchPageFrom(page searchresult.Page) searchPage {
	items := make([]searchItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, searchItemFrom(item))
	}

	return searchPage{Channels: []searchChannel{{
		Title:        channelTitle,
		Items:        items,
		TotalResults: strconv.Itoa(len(items)),
		StartIndex:   "0",
		ItemsPerPage: strconv.Itoa(len(items)),
	}}}
}

func searchItemFrom(item searchresult.Item) searchItem {
	published := ""
	if instant, ok := item.PublishedAt.Get(); ok {
		published = instant.Format(time.RFC1123Z)
	}

	return searchItem{
		Title:       item.Title,
		Link:        item.Address,
		Description: item.Description,
		PubDate:     published,
		Image:       item.ImageAddress,
	}
}
