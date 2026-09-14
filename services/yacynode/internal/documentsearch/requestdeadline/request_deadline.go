// Package requestdeadline reports whether the request a search serves has
// ended. A search asks it while it reads, so it answers the peer within the
// time the peer grants.
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
