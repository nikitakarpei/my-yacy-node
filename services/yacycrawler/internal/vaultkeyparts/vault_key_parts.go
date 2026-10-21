// Package vaultkeyparts carries the vault key parts for crawl IDs and URL hashes,
// so every crawl state record that keys on them shares one byte form and one parser.
package vaultkeyparts

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

var errKeyPartLength = errors.New("key part has the wrong length")

var CrawlID = vault.BytesKeyPartFrom(appendCrawlID, fixedBytesFrom[crawlid.CrawlID])

var URLHash = vault.BytesKeyPartFrom(appendURLHash, fixedBytesFrom[urlhash.URLHash])

func appendCrawlID(id crawlid.CrawlID, buffer []byte) []byte {
	return append(buffer, id[:]...)
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
