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
	shortfallOrderBucket vault.Name = "rwidistribution_offer_shortfall_order"
	refreshOrderBucket   vault.Name = "rwidistribution_offer_refresh_order"
	offerPlaceBucket     vault.Name = "rwidistribution_offer_place"
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

func registerOfferPlaces(
	v *vault.Vault,
) (*vault.Collection[postingidentity.Identity, offerPlace], error) {
	offerPlaces, err := v.RegisterCollection(
		offerPlaceBucket,
		postingidentity.KeyLayout,
		offerPlaceValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register offer place: %w", err)
	}

	return offerPlaces, nil
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
		return sectorKeyOf(
			offer.Place.Sector,
		), offer.Place.DueAt, offer.Identity.Word, offer.Identity.URL
	},
	func(
		sector int64,
		dueAt time.Time,
		word yacymodel.Hash,
		url yacymodel.URLHash,
	) scheduledPostingOffer {
		return scheduledPostingOffer{
			Place:    offerPlace{Sector: yacymodel.DHTRingSector(sector), DueAt: dueAt},
			Identity: postingidentity.Identity{Word: word, URL: url},
		}
	},
)

func sectorKeyOf(sector yacymodel.DHTRingSector) int64 {
	return int64(sector) //nolint:gosec // a sector is at most MaxDHTRingSector
}

func everyOfferInSector(sector yacymodel.DHTRingSector) vault.KeyRange {
	return orderKeyParts.KeysWithFirst(sectorKeyOf(sector))
}

func everyOfferInSectorDueBy(sector yacymodel.DHTRingSector, dueAt time.Time) vault.KeyRange {
	return orderKeyParts.KeysWithFirstThroughSecond(sectorKeyOf(sector), dueAt)
}

var (
	errBadOfferPlace    = errors.New("bad offer place")
	errBadOfferInterval = errors.New("bad offer interval")
)

type offerPlaceValueCodec struct{}

func (offerPlaceValueCodec) Encode(place offerPlace) ([]byte, error) {
	var stored storedfields.Writer
	stored.Count(int(place.Sector))
	stored.Time(place.DueAt)

	return stored.Record(), nil
}

func (offerPlaceValueCodec) Decode(raw []byte) (offerPlace, error) {
	stored := storedfields.ReaderOf(raw, errBadOfferPlace)
	sector := stored.Count("sector")
	dueAt := stored.Time("due at")

	return offerPlace{Sector: yacymodel.DHTRingSector(sector), DueAt: dueAt}, stored.Err()
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
