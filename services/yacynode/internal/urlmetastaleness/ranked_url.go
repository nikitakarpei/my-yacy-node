package urlmetastaleness

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var orderKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

// freshnessRank orders days as plain text so the vault's byte order is also
// stalest-first order. A url with no known day ranks stalest.
type freshnessRank string

func rankOf(day yacymodel.Optional[yacymodel.CalendarDay]) freshnessRank {
	value, _ := day.Get()

	return freshnessRank(fmt.Sprintf("%04d%02d%02d", value.Year, value.Month, value.Day))
}

type rankedURL struct {
	rank freshnessRank
	hash yacymodel.URLHash
}

type orderKeyCodec struct{}

func (orderKeyCodec) Encode(ranked rankedURL) vaultkey.Key {
	return orderKeyLayout.Key(string(ranked.rank), ranked.hash.String())
}

func (orderKeyCodec) Decode(key vaultkey.Key) (rankedURL, error) {
	rank, hash, err := orderKeyLayout.Parts(key)
	if err != nil {
		return rankedURL{}, fmt.Errorf("staleness order key: %w", err)
	}

	parsedHash, err := yacymodel.ParseURLHash(hash)
	if err != nil {
		return rankedURL{}, fmt.Errorf("staleness order url hash: %w", err)
	}

	return rankedURL{rank: freshnessRank(rank), hash: parsedHash}, nil
}
