package rwiescrow

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type heldPosting struct {
	HeldAt  time.Time
	Posting yacymodel.RWIPosting
}

type heldPostingCodec struct{}

func (heldPostingCodec) Encode(held heldPosting) ([]byte, error) {
	raw, err := rwipostings.PostingForm().Encode(held.Posting)
	if err != nil {
		return nil, fmt.Errorf("encode held posting: %w", err)
	}

	return append(heldAtPrefix(held.HeldAt), raw...), nil
}

func (heldPostingCodec) Decode(raw []byte) (heldPosting, error) {
	if len(raw) < heldAtDigits {
		return heldPosting{}, fmt.Errorf(
			"held posting value: length %d, want at least %d",
			len(raw),
			heldAtDigits,
		)
	}

	var nanos int64
	if _, err := fmt.Sscanf(string(raw[:heldAtDigits]), "%d", &nanos); err != nil {
		return heldPosting{}, fmt.Errorf("held posting hold time: %w", err)
	}

	posting, err := rwipostings.PostingForm().Decode(raw[heldAtDigits:])
	if err != nil {
		return heldPosting{}, fmt.Errorf("decode held posting: %w", err)
	}

	return heldPosting{HeldAt: time.Unix(0, nanos), Posting: posting}, nil
}
