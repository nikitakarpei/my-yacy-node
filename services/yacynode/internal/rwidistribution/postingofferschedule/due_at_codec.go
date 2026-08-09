package postingofferschedule

import (
	"encoding/binary"
	"fmt"
	"time"
)

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
