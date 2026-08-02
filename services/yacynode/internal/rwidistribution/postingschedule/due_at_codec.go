package postingschedule

import (
	"fmt"
	"time"
)

type dueAtCodec struct{}

func (dueAtCodec) Encode(at time.Time) ([]byte, error) {
	return fmt.Appendf(nil, "%0*d", dueAtDigits, at.UnixNano()), nil
}

func (dueAtCodec) Decode(raw []byte) (time.Time, error) {
	var nanos int64
	if _, err := fmt.Sscanf(string(raw), "%d", &nanos); err != nil {
		return time.Time{}, fmt.Errorf("due at: %w", err)
	}

	return time.Unix(0, nanos), nil
}
