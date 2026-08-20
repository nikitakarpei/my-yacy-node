package postingofferschedule

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
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
	order, err := vault.RegisterSet(v, orderBucket, orderKeyCodec{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("register offer order: %w", err)
	}
	dueTimes, err := vault.RegisterCollection(
		v,
		dueBucket,
		postingidentity.KeyCodec{},
		dueAtValueCodec{},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("register offer due: %w", err)
	}
	offerIntervals, err := vault.RegisterCollection(
		v,
		offerIntervalBucket,
		postingidentity.KeyCodec{},
		offerIntervalValueCodec{},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("register offer interval: %w", err)
	}

	return order, dueTimes, offerIntervals, nil
}

var orderKeyLayout = vault.TripleKey(vault.TimeKeyPart, hashcodec.Hash, hashcodec.URLHash)

type orderKeyCodec struct{}

func (orderKeyCodec) Encode(offer scheduledPostingOffer) vault.Key {
	return orderKeyLayout.Key(offer.At, offer.Posting.Word, offer.Posting.URL)
}

func (orderKeyCodec) Decode(storedKey []byte) (scheduledPostingOffer, error) {
	dueAt, word, url, err := orderKeyLayout.Parts(storedKey)
	if err != nil {
		return scheduledPostingOffer{}, fmt.Errorf("offer schedule order key: %w", err)
	}

	return scheduledPostingOffer{At: dueAt, Posting: postingidentity.IdentityOf(word, url)}, nil
}

func everyOfferDueBy(dueAt time.Time) vault.KeyRange {
	return orderKeyLayout.KeysThroughFirst(dueAt)
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
