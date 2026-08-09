package postingofferschedule

import (
	"fmt"
	"time"
)

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
