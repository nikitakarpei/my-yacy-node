package observedfetch

import "time"

type Clock interface {
	Now() time.Time
}
