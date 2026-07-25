package frontier

import "time"

type PendingVisit struct {
	URL       string
	Depth     int
	deferrals int
	attempts  int
	notBefore time.Time
}
