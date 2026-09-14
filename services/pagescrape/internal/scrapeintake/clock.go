package scrapeintake

import "time"

type Clock interface {
	Now() time.Time
}
