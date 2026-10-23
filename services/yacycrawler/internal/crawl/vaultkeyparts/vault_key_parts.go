// Package vaultkeyparts carries the vault key parts for crawl IDs, in start order or newest
// first, and URL hashes, so every crawl state record that keys on them shares one byte form
// and one parser.
package vaultkeyparts

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/urlhash"
)

var errKeyPartLength = errors.New("key part has the wrong length")

var CrawlID = vault.BytesKeyPartFrom(appendCrawlID, fixedBytesFrom[crawlid.CrawlID])

var CrawlIDNewestFirst = vault.BytesKeyPartFrom(appendInvertedCrawlID, invertedCrawlIDFrom)

var URLHash = vault.BytesKeyPartFrom(appendURLHash, fixedBytesFrom[urlhash.URLHash])

func appendCrawlID(id crawlid.CrawlID, buffer []byte) []byte {
	return append(buffer, id[:]...)
}

func appendInvertedCrawlID(id crawlid.CrawlID, buffer []byte) []byte {
	for _, idByte := range id {
		buffer = append(buffer, ^idByte)
	}
	return buffer
}

func invertedCrawlIDFrom(stored vault.StoredBytes) (crawlid.CrawlID, error) {
	inverted, err := fixedBytesFrom[crawlid.CrawlID](stored)
	for position := range inverted {
		inverted[position] = ^inverted[position]
	}
	return inverted, err
}

func appendURLHash(hash urlhash.URLHash, buffer []byte) []byte {
	return append(buffer, hash[:]...)
}

func fixedBytesFrom[A ~[16]byte](stored vault.StoredBytes) (A, error) {
	var parsed A
	raw := stored.AppendTo(nil)
	if len(raw) != len(parsed) {
		return parsed, fmt.Errorf("%w: %d bytes, want %d", errKeyPartLength, len(raw), len(parsed))
	}
	copy(parsed[:], raw)
	return parsed, nil
}
