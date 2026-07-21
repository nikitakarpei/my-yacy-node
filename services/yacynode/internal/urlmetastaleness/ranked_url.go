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

func rankOf(day yacymodel.CalendarDay) freshnessRank {
	return freshnessRank(fmt.Sprintf("%04d%02d%02d", day.Year, day.Month, day.Day))
}

type rankedURL struct {
	rank freshnessRank
	hash yacymodel.Hash
}

func (r rankedURL) orderKey() vault.Key {
	var key bytes.Buffer
	key.WriteString(string(r.rank))
	key.WriteByte(freshnessHashSeparator)
	key.WriteString(r.hash.String())

	return key.Bytes()
}

func hashFromOrderKey(key vault.Key) (yacymodel.Hash, error) {
	_, encodedHash, found := bytes.Cut(key, []byte{freshnessHashSeparator})
	if !found {
		return yacymodel.Hash{}, fmt.Errorf("staleness order key without separator")
	}

	hash, err := yacymodel.ParseHash(string(encodedHash))
	if err != nil {
		return yacymodel.Hash{}, fmt.Errorf("staleness order hash: %w", err)
	}

	return hash, nil
}
