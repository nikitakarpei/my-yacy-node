package jetstream

import "time"

type Clock interface {
	Now() time.Time
}
