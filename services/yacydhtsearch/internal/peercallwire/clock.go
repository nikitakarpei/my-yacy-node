package peercallwire

import "time"

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}
