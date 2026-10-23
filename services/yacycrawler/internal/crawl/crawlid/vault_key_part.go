package crawlid

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var errKeyPartLength = errors.New("crawl ID key part has the wrong length")

var KeyPart = vault.BytesKeyPartFrom(appendCrawlID, crawlIDFrom)

var KeyPartNewestFirst = vault.BytesKeyPartFrom(appendInvertedCrawlID, invertedCrawlIDFrom)

func appendCrawlID(id CrawlID, buffer []byte) []byte {
	return append(buffer, id[:]...)
}

func crawlIDFrom(stored vault.StoredBytes) (CrawlID, error) {
	var id CrawlID
	raw := stored.AppendTo(nil)
	if len(raw) != len(id) {
		return id, fmt.Errorf("%w: %d bytes, want %d", errKeyPartLength, len(raw), len(id))
	}
	copy(id[:], raw)
	return id, nil
}

func appendInvertedCrawlID(id CrawlID, buffer []byte) []byte {
	for _, idByte := range id {
		buffer = append(buffer, ^idByte)
	}
	return buffer
}

func invertedCrawlIDFrom(stored vault.StoredBytes) (CrawlID, error) {
	inverted, err := crawlIDFrom(stored)
	for position := range inverted {
		inverted[position] = ^inverted[position]
	}
	return inverted, err
}
