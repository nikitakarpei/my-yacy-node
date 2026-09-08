package pagemarkdownstore

import "time"

const (
	RecalledPagePath       = "/page/markdown"
	RequestedURLQueryParam = "url"
)

type RecalledPage struct {
	CanonicalURL string    `json:"canonicalUrl"`
	Markdown     string    `json:"markdown"`
	StoredAt     time.Time `json:"storedAt"`
	Version      string    `json:"version"`
}
