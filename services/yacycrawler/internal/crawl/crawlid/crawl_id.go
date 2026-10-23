// Package crawlid names one crawl the crawler runs for an accepted order: a version 7
// UUID the crawler mints when it keeps the order, so crawl IDs sort by the time each
// crawl began. Its vault key parts sort crawls in that order or newest first.
package crawlid

import (
	"fmt"

	"github.com/google/uuid"
)

type CrawlID [16]byte

func New() (CrawlID, error) {
	minted, err := uuid.NewV7()
	if err != nil {
		return CrawlID{}, fmt.Errorf("mint crawl ID: %w", err)
	}
	return CrawlID(minted), nil
}

func (id CrawlID) String() string {
	return uuid.UUID(id).String()
}

func (id CrawlID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *CrawlID) UnmarshalText(text []byte) error {
	parsed, err := uuid.ParseBytes(text)
	if err != nil {
		return fmt.Errorf("read crawl ID %q: %w", text, err)
	}
	*id = CrawlID(parsed)
	return nil
}
