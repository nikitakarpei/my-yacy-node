package rwiescrow

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var expiryKeyLayout = vaultkey.Triple(vaultkey.Time, vaultkey.Text, vaultkey.Text)

type postingHold struct {
	HeldAt  time.Time
	Posting postingIdentity
}

type expiryKeyCodec struct{}

func (expiryKeyCodec) Encode(hold postingHold) vaultkey.Key {
	return expiryKeyLayout.Key(hold.HeldAt, hold.Posting.Word.String(), hold.Posting.URL.String())
}

func (expiryKeyCodec) Decode(key vaultkey.Key) (postingHold, error) {
	heldAt, word, url, err := expiryKeyLayout.Parts(key)
	if err != nil {
		return postingHold{}, fmt.Errorf("held posting expiry key: %w", err)
	}
	posting, err := parsedIdentity(word, url)
	if err != nil {
		return postingHold{}, err
	}

	return postingHold{HeldAt: heldAt, Posting: posting}, nil
}

func expiryKeysBefore(cutoff time.Time) vaultkey.KeyRange {
	return vaultkey.KeysBefore(expiryKeyLayout.First(cutoff))
}
