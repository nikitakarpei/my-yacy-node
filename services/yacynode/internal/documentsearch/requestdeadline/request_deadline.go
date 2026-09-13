// Package requestdeadline reports whether the request a search serves has
// ended. A read of the index asks between postings, so it stops and answers
// with the documents it holds instead of running past the time the peer grants.
package requestdeadline

import "context"

func RequestHasEnded(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
