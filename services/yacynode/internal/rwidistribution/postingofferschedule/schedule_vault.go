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
	dueBucket            vault.Name = "rwidistribution_offer_due"
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
) (*vault.Collection[postingidentity.Identity, time.Time], error) {
	dueTimes, err := v.RegisterCollection(
		dueBucket,
		postingidentity.KeyLayout,
		dueAtValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register offer due: %w", err)
	}

	return dueTimes, nil
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

var orderKeyParts = vault.TripleKey(vault.TimeKeyPart, hashkeypart.Hash, hashkeypart.URLHash)

var orderKeyLayout = orderKeyParts.KeyLayoutFor(
	func(offer scheduledPostingOffer) (time.Time, yacymodel.Hash, yacymodel.URLHash) {
		return offer.At, offer.Identity.Word, offer.Identity.URL
	},
	func(dueAt time.Time, word yacymodel.Hash, url yacymodel.URLHash) scheduledPostingOffer {
		return scheduledPostingOffer{
			At:       dueAt,
			Identity: postingidentity.Identity{Word: word, URL: url},
		}
	},
)

func everyOfferDueBy(dueAt time.Time) vault.KeyRange {
	return orderKeyParts.KeysThroughFirst(dueAt)
}

var (
	errBadDueTime       = errors.New("bad offer due time")
	errBadOfferInterval = errors.New("bad offer interval")
)

type dueAtValueCodec struct{}

func (dueAtValueCodec) Encode(at time.Time) ([]byte, error) {
	var stored storedfields.Writer
	stored.Time(at)

	return stored.Record(), nil
}

func (dueAtValueCodec) Decode(raw []byte) (time.Time, error) {
	stored := storedfields.ReaderOf(raw, errBadDueTime)
	dueAt := stored.Time("due at")

	return dueAt, stored.Err()
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
