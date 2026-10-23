package robotsexclusion

import "time"

type Verdict int

const (
	Deferred Verdict = iota
	Allowed
	Disallowed
)

type FetchPermission struct {
	Verdict  Verdict
	DeferFor time.Duration
}
