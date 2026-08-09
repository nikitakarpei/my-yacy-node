package rwiescrow

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type heldPosting struct {
	HeldAt  time.Time
	Posting yacymodel.RWIPosting
}

type heldPostingValueCodec struct{}

func (heldPostingValueCodec) Encode(held heldPosting) ([]byte, error) {
	raw, err := rwipostings.PostingForm().Encode(held.Posting)
	if err != nil {
		return nil, fmt.Errorf("encode held posting: %w", err)
	}

	heldAt, err := binary.Append(nil, binary.BigEndian, []int64{
		held.HeldAt.Unix(),
		int64(held.HeldAt.Nanosecond()),
	})
	if err != nil {
		return nil, fmt.Errorf("encode held posting hold time: %w", err)
	}

	return append(heldAt, raw...), nil
}

func (heldPostingValueCodec) Decode(raw []byte) (heldPosting, error) {
	heldAt := make([]int64, 2)
	heldAtLength, err := binary.Decode(raw, binary.BigEndian, heldAt)
	if err != nil {
		return heldPosting{}, fmt.Errorf("held posting hold time: %w", err)
	}
	seconds, nanoseconds := heldAt[0], heldAt[1]

	posting, err := rwipostings.PostingForm().Decode(raw[heldAtLength:])
	if err != nil {
		return heldPosting{}, fmt.Errorf("decode held posting: %w", err)
	}

	return heldPosting{HeldAt: time.Unix(seconds, nanoseconds).UTC(), Posting: posting}, nil
}
