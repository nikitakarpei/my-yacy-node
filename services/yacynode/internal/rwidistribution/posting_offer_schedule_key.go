package rwidistribution

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const dueAtDigits = 20

type scheduledPostingOffer struct {
	At      time.Time
	Posting duePosting
}

func postingKey(word yacymodel.Hash, url yacymodel.URLHash) vault.Key {
	key := make(vault.Key, 0, yacymodel.HashLength*2)
	key = append(key, word.String()...)
	key = append(key, url.String()...)

	return key
}

func orderKey(at time.Time, word yacymodel.Hash, url yacymodel.URLHash) vault.Key {
	key := make(vault.Key, 0, dueAtDigits+yacymodel.HashLength*2)
	key = fmt.Appendf(key, "%0*d", dueAtDigits, at.UnixNano())
	key = append(key, word.String()...)
	key = append(key, url.String()...)

	return key
}

func parseOrderKey(key vault.Key) (scheduledPostingOffer, error) {
	wantLen := dueAtDigits + yacymodel.HashLength*2
	if len(key) != wantLen {
		return scheduledPostingOffer{},
			fmt.Errorf("offer schedule order key: length %d, want %d", len(key), wantLen)
	}

	var nanos int64
	if _, err := fmt.Sscanf(string(key[:dueAtDigits]), "%d", &nanos); err != nil {
		return scheduledPostingOffer{},
			fmt.Errorf("offer schedule order key: parse due time: %w", err)
	}

	word, err := yacymodel.ParseHash(string(key[dueAtDigits : dueAtDigits+yacymodel.HashLength]))
	if err != nil {
		return scheduledPostingOffer{},
			fmt.Errorf("offer schedule order key: word hash: %w", err)
	}

	url, err := yacymodel.ParseURLHash(string(key[dueAtDigits+yacymodel.HashLength:]))
	if err != nil {
		return scheduledPostingOffer{},
			fmt.Errorf("offer schedule order key: url hash: %w", err)
	}

	return scheduledPostingOffer{
		At:      time.Unix(0, nanos),
		Posting: duePosting{Word: word, URL: url},
	}, nil
}
