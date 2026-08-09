package postingofferschedule

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var orderKeyLayout = vaultkey.Triple(vaultkey.Time, vaultkey.Text, vaultkey.Text)

type scheduledPostingOffer struct {
	At      time.Time
	Posting postingidentity.Identity
}

type orderKeyCodec struct{}

func (orderKeyCodec) Encode(offer scheduledPostingOffer) vaultkey.Key {
	return orderKeyLayout.Key(offer.At, offer.Posting.Word.String(), offer.Posting.URL.String())
}

func (orderKeyCodec) Decode(key vaultkey.Key) (scheduledPostingOffer, error) {
	dueAt, word, url, err := orderKeyLayout.Parts(key)
	if err != nil {
		return scheduledPostingOffer{}, fmt.Errorf("offer schedule order key: %w", err)
	}

	parsedWord, err := yacymodel.ParseHash(word)
	if err != nil {
		return scheduledPostingOffer{},
			fmt.Errorf("offer schedule order key: word hash: %w", err)
	}

	parsedURL, err := yacymodel.ParseURLHash(url)
	if err != nil {
		return scheduledPostingOffer{},
			fmt.Errorf("offer schedule order key: url hash: %w", err)
	}

	return scheduledPostingOffer{
		At:      dueAt,
		Posting: postingidentity.IdentityOf(parsedWord, parsedURL),
	}, nil
}

func orderKeysThrough(dueAt time.Time) vaultkey.KeyRange {
	return vaultkey.KeysThrough(orderKeyLayout.First(dueAt))
}
