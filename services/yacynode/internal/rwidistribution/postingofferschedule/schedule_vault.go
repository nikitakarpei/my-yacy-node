package postingofferschedule

import (
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
)

const (
	shortfallOrderBucket vault.Name = "rwidistribution_offer_shortfall_sector_order"
	refreshOrderBucket   vault.Name = "rwidistribution_offer_refresh_sector_order"
	dueBucket            vault.Name = "rwidistribution_offer_sector_due"
	offerIntervalBucket  vault.Name = "rwidistribution_offer_interval"
)

func registerOfferOrder(
	v *vault.Vault,
	bucket vault.Name,
) (*vault.Set[scheduledPostingOffer], error) {
	order, err := v.RegisterSet(bucket, orderKeyLayout)
	if err != nil {
		return nil, fmt.Errorf("register offer order %s: %w", bucket, err)
	}

	return order, nil
}

func registerOfferDues(
	v *vault.Vault,
) (*vault.Collection[postingidentity.Identity, offerDue], error) {
	offerDues, err := v.RegisterCollection(
		dueBucket,
		postingidentity.KeyLayout,
		offerDueValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register offer due: %w", err)
	}

	return offerDues, nil
}

func registerOfferIntervals(
	v *vault.Vault,
) (*vault.Collection[postingidentity.Identity, time.Duration], error) {
	offerIntervals, err := v.RegisterCollection(
		offerIntervalBucket,
		postingidentity.KeyLayout,
		offerIntervalValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register offer interval: %w", err)
	}

	return offerIntervals, nil
}

var orderKeyParts = vault.QuadKey(
	vault.IntegerKeyPart,
	vault.TimeKeyPart,
	hashkeypart.Hash,
	hashkeypart.URLHash,
)

var orderKeyLayout = orderKeyParts.KeyLayoutFor(
	func(offer scheduledPostingOffer) (int64, time.Time, yacymodel.Hash, yacymodel.URLHash) {
		return sectorKeyOf(offer.Due.Sector), offer.Due.At, offer.Identity.Word, offer.Identity.URL
	},
	func(
		sector int64,
		dueAt time.Time,
		word yacymodel.Hash,
		url yacymodel.URLHash,
	) scheduledPostingOffer {
		return scheduledPostingOffer{
			Due:      offerDue{Sector: yacymodel.DHTRingSector(sector), At: dueAt},
			Identity: postingidentity.Identity{Word: word, URL: url},
		}
	},
)

func sectorKeyOf(sector yacymodel.DHTRingSector) int64 {
	return int64(sector) //nolint:gosec // a sector is at most MaxDHTRingSector
}

func everyOfferIn(sector yacymodel.DHTRingSector) vault.KeyRange {
	return orderKeyParts.KeysWithFirst(sectorKeyOf(sector))
}

func everyOfferInSectorDueBy(sector yacymodel.DHTRingSector, dueAt time.Time) vault.KeyRange {
	return orderKeyParts.KeysWithFirstThroughSecond(sectorKeyOf(sector), dueAt)
}

var (
	errBadOfferDue      = errors.New("bad offer due")
	errBadOfferInterval = errors.New("bad offer interval")
)

type offerDueValueCodec struct{}

func (offerDueValueCodec) Encode(due offerDue) ([]byte, error) {
	var stored storedfields.Writer
	stored.Count(int(due.Sector))
	stored.Time(due.At)

	return stored.Record(), nil
}

func (offerDueValueCodec) Decode(raw []byte) (offerDue, error) {
	stored := storedfields.ReaderOf(raw, errBadOfferDue)
	sector := stored.Count("due sector")
	dueAt := stored.Time("due at")

	return offerDue{Sector: yacymodel.DHTRingSector(sector), At: dueAt}, stored.Err()
}

type offerIntervalValueCodec struct{}

func (offerIntervalValueCodec) Encode(interval time.Duration) ([]byte, error) {
	var stored storedfields.Writer
	stored.Varint(int64(interval.Seconds()))

	return stored.Record(), nil
}

func (offerIntervalValueCodec) Decode(raw []byte) (time.Duration, error) {
	stored := storedfields.ReaderOf(raw, errBadOfferInterval)
	seconds := stored.Varint("offer interval")

	return time.Duration(seconds) * time.Second, stored.Err()
}
