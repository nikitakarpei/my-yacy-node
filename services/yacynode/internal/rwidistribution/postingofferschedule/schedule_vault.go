package postingofferschedule

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
)

const (
	orderBucket         vault.Name = "rwidistribution_offer_order"
	dueBucket           vault.Name = "rwidistribution_offer_due"
	offerIntervalBucket vault.Name = "rwidistribution_offer_interval"
)

func registerSchedule(v *vault.Vault) (
	*vault.Set[scheduledPostingOffer],
	*vault.Collection[postingidentity.Identity, time.Time],
	*vault.Collection[postingidentity.Identity, time.Duration],
	error,
) {
	order, err := v.RegisterSet(orderBucket, orderKeyLayout)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("register offer order: %w", err)
	}
	dueTimes, err := v.RegisterCollection(
		dueBucket,
		postingidentity.KeyLayout,
		dueAtValueCodec{},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("register offer due: %w", err)
	}
	offerIntervals, err := v.RegisterCollection(
		offerIntervalBucket,
		postingidentity.KeyLayout,
		offerIntervalValueCodec{},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("register offer interval: %w", err)
	}

	return order, dueTimes, offerIntervals, nil
}

var orderKeyParts = vault.TripleKey(vault.TimeKeyPart, hashkeypart.Hash, hashkeypart.URLHash)

var orderKeyLayout = orderKeyParts.KeyLayoutFor(
	func(offer scheduledPostingOffer) (time.Time, yacymodel.Hash, yacymodel.URLHash) {
		return offer.At, offer.Posting.Word, offer.Posting.URL
	},
	func(dueAt time.Time, word yacymodel.Hash, url yacymodel.URLHash) scheduledPostingOffer {
		return scheduledPostingOffer{At: dueAt, Posting: postingidentity.IdentityOf(word, url)}
	},
)

func everyOfferDueBy(dueAt time.Time) vault.KeyRange {
	return orderKeyParts.KeysThroughFirst(dueAt)
}

type dueAtValueCodec struct{}

func (dueAtValueCodec) Encode(at time.Time) ([]byte, error) {
	raw, err := binary.Append(nil, binary.BigEndian, []int64{
		at.Unix(),
		int64(at.Nanosecond()),
	})
	if err != nil {
		return nil, fmt.Errorf("encode due at: %w", err)
	}

	return raw, nil
}

func (dueAtValueCodec) Decode(raw []byte) (time.Time, error) {
	dueAt := make([]int64, 2)
	if _, err := binary.Decode(raw, binary.BigEndian, dueAt); err != nil {
		return time.Time{}, fmt.Errorf("due at: %w", err)
	}
	seconds, nanoseconds := dueAt[0], dueAt[1]

	return time.Unix(seconds, nanoseconds).UTC(), nil
}

type offerIntervalValueCodec struct{}

func (offerIntervalValueCodec) Encode(interval time.Duration) ([]byte, error) {
	return fmt.Appendf(nil, "%d", interval.Nanoseconds()), nil
}

func (offerIntervalValueCodec) Decode(raw []byte) (time.Duration, error) {
	var nanos int64
	if _, err := fmt.Sscanf(string(raw), "%d", &nanos); err != nil {
		return 0, fmt.Errorf("offer interval: %w", err)
	}

	return time.Duration(nanos), nil
}
