package postingofferwait

import (
	"fmt"
	"time"
)

type waitCodec struct{}

func (waitCodec) Encode(wait time.Duration) ([]byte, error) {
	return fmt.Appendf(nil, "%d", wait.Nanoseconds()), nil
}

func (waitCodec) Decode(raw []byte) (time.Duration, error) {
	var nanos int64
	if _, err := fmt.Sscanf(string(raw), "%d", &nanos); err != nil {
		return 0, fmt.Errorf("offer interval: %w", err)
	}

	return time.Duration(nanos), nil
}
