package urlmetastaleness

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
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

func (r rankedURL) orderKey() vault.Key {
	return orderKeyLayout.Key(string(r.rank), r.hash.String()).Bytes()
}

func hashFromOrderKey(key vault.Key) (yacymodel.URLHash, error) {
	_, hash, err := orderKeyLayout.Parts(vaultkey.KeyFrom(key))
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("staleness order key: %w", err)
	}

	return yacymodel.ParseURLHash(hash)
}
