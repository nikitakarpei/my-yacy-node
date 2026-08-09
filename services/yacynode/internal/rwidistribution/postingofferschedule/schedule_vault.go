package postingofferschedule

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
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

var orderKeyLayout = vaultkey.Triple(vaultkey.Time, vaultkey.Text, vaultkey.Text)

type orderKeyCodec struct{}

func (orderKeyCodec) Encode(offer scheduledPostingOffer) vaultkey.Key {
	return orderKeyLayout.Key(offer.At, offer.Posting.Word.String(), offer.Posting.URL.String())
}

func (orderKeyCodec) Decode(key vaultkey.Key) (scheduledPostingOffer, error) {
	dueAt, word, url, err := orderKeyLayout.Parts(key)
	if err != nil {
		return scheduledPostingOffer{}, fmt.Errorf("offer schedule order key: %w", err)
	}

	parsedWord, err := yacymodel.ParseHash(word)
	if err != nil {
		return scheduledPostingOffer{},
			fmt.Errorf("offer schedule order key: word hash: %w", err)
	}

	parsedURL, err := yacymodel.ParseURLHash(url)
	if err != nil {
		return scheduledPostingOffer{},
			fmt.Errorf("offer schedule order key: url hash: %w", err)
	}

	return scheduledPostingOffer{
		At:      dueAt,
		Posting: postingidentity.IdentityOf(parsedWord, parsedURL),
	}, nil
}

func orderKeysThrough(dueAt time.Time) vaultkey.KeyRange {
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
