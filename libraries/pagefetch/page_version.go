package pagefetch

import "time"

type PageVersion struct {
	EntityTag  string
	ModifiedAt time.Time
}
