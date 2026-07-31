package urlmetastaleness

import (
	"bytes"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const freshnessHashSeparator = 0x00

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
	var key bytes.Buffer
	key.WriteString(string(r.rank))
	key.WriteByte(freshnessHashSeparator)
	key.WriteString(r.hash.String())

	return key.Bytes()
}

func hashFromOrderKey(key vault.Key) (yacymodel.URLHash, error) {
	_, encodedHash, found := bytes.Cut(key, []byte{freshnessHashSeparator})
	if !found {
		return yacymodel.URLHash{}, fmt.Errorf("staleness order key without separator")
	}

	hash, err := yacymodel.ParseURLHash(string(encodedHash))
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("staleness order hash: %w", err)
	}

	return hash, nil
}
