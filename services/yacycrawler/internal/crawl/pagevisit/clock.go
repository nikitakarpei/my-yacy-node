package pagevisit

import "time"

type Clock interface {
	Now() time.Time
}
