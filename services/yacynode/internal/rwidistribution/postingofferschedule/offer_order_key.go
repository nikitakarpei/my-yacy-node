package postingofferschedule

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var orderKeyLayout = vaultkey.Triple(vaultkey.Time, vaultkey.Text, vaultkey.Text)

type scheduledPostingOffer struct {
	At      time.Time
	Posting postingidentity.Identity
}

func orderKeyFor(posting postingidentity.Identity, dueAt time.Time) vault.Key {
	return orderKeyLayout.Key(dueAt, posting.Word.String(), posting.URL.String()).Bytes()
}

func parseOrderKey(key vault.Key) (scheduledPostingOffer, error) {
	dueAt, word, url, err := orderKeyLayout.Parts(vaultkey.KeyFrom(key))
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
