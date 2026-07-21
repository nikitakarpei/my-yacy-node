package firecrawlscrape

type scrapeRequest struct {
	URL     string   `json:"url"`
	Formats []string `json:"formats,omitempty"`
}

type scrapeResponse struct {
	Success bool        `json:"success"`
	Data    *scrapeData `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type scrapeData struct {
	Markdown string         `json:"markdown,omitempty"`
	Metadata scrapeMetadata `json:"metadata"`
}

type scrapeMetadata struct {
	Title     string `json:"title,omitempty"`
	Language  string `json:"language,omitempty"`
	SourceURL string `json:"sourceURL,omitempty"`
}
