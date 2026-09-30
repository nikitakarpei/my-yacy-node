// Package proxiedrequest holds a request that the proxy relays to the origin,
// with its response headers deadline.
package proxiedrequest

import (
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Request struct {
	Method          string
	Address         canonicalurl.CanonicalURL
	Headers         http.Header
	HeadersDeadline time.Time
}
