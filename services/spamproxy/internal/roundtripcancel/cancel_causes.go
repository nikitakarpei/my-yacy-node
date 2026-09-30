// Package roundtripcancel names why a round trip was cancelled.
package roundtripcancel

import "errors"

var (
	ErrHeadersDeadlinePassed = errors.New("response headers deadline passed")
	ErrIdleTimeoutPassed     = errors.New("upstream response body idle timeout passed")
)
