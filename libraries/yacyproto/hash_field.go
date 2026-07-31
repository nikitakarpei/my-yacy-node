package yacyproto

import (
	"fmt"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func parseHashField(scope, field, raw string) (yacymodel.Hash, error) {
	if raw == "" {
		return yacymodel.Hash{}, nil
	}

	hash, err := yacymodel.ParseHash(raw)
	if err != nil {
		return yacymodel.Hash{}, fmt.Errorf("%s %s: %w", scope, field, err)
	}

	return hash, nil
}

func concatHashes(hashes []yacymodel.Hash) string {
	if len(hashes) == 0 {
		return ""
	}

	var b strings.Builder
	for _, h := range hashes {
		b.WriteString(h.String())
	}

	return b.String()
}

func splitSearchHashes(field, raw string) ([]yacymodel.Hash, error) {
	var hashes []yacymodel.Hash
	for i := 0; i+yacymodel.HashLength <= len(raw); i += yacymodel.HashLength {
		hash, err := parseHashField("search request", field, raw[i:i+yacymodel.HashLength])
		if err != nil {
			return nil, err
		}

		hashes = append(hashes, hash)
	}

	return hashes, nil
}

func joinURLHashes(urls []yacymodel.URLHash) string {
	if len(urls) == 0 {
		return ""
	}

	parts := make([]string, len(urls))
	for i, u := range urls {
		parts[i] = u.String()
	}

	return strings.Join(parts, ",")
}

func splitURLHashes(scope, field, raw string) ([]yacymodel.URLHash, error) {
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	urls := make([]yacymodel.URLHash, 0, len(parts))
	for _, part := range parts {
		url, err := yacymodel.ParseURLHash(part)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", scope, field, err)
		}

		urls = append(urls, url)
	}

	return urls, nil
}

func concatURLHashes(urls []yacymodel.URLHash) string {
	if len(urls) == 0 {
		return ""
	}

	var b strings.Builder
	for _, u := range urls {
		b.WriteString(u.String())
	}

	return b.String()
}

func splitSearchURLHashes(field, raw string) ([]yacymodel.URLHash, error) {
	var urls []yacymodel.URLHash
	for i := 0; i+yacymodel.HashLength <= len(raw); i += yacymodel.HashLength {
		url, err := yacymodel.ParseURLHash(raw[i : i+yacymodel.HashLength])
		if err != nil {
			return nil, fmt.Errorf("search request %s: %w", field, err)
		}

		urls = append(urls, url)
	}

	return urls, nil
}
