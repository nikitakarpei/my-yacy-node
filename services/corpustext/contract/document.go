package searchdocument

import "time"

type Document struct {
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	Content       string    `json:"content"`
	AuxiliaryText string    `json:"auxiliary_text"`
	CrawledAt     time.Time `json:"crawled_at"`
	Language      string    `json:"language"`
}
