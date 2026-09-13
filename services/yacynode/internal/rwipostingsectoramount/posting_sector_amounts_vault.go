package rwipostingsectoramount

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const postingSectorAmountBucket vault.Name = "rwi_sector_amount"

func registerPostingSectorAmounts(
	v *vault.Vault,
) (*vault.Collection[yacymodel.DHTRingSector, int], error) {
	amountPerSector, err := v.RegisterCollection(
		postingSectorAmountBucket,
		dhtRingSectorKeyLayout,
		postingSectorAmountValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register posting amounts per sector: %w", err)
	}

	return amountPerSector, nil
}

var dhtRingSectorKeyLayout = vault.SingleKey(vault.IntegerKeyPart).KeyLayoutFor(
	func(sector yacymodel.DHTRingSector) int64 { return int64(sector) },
	func(sector int64) yacymodel.DHTRingSector { return yacymodel.DHTRingSector(sector) },
)

var errBadPostingSectorAmount = errors.New("bad posting amount of sector")

type postingSectorAmountValueCodec struct{}

func (postingSectorAmountValueCodec) Encode(amountOfPostings int) ([]byte, error) {
	var stored storedfields.Writer
	stored.Count(amountOfPostings)

	return stored.Record(), nil
}

func (postingSectorAmountValueCodec) Decode(raw []byte) (int, error) {
	stored := storedfields.ReaderOf(raw, errBadPostingSectorAmount)
	amountOfPostings := stored.Count("posting amount of sector")

	return amountOfPostings, stored.Err()
}
