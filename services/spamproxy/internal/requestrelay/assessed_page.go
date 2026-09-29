package requestrelay

import (
	"net/http"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	spamassessmenthttpheader "github.com/nikitakarpei/yacy-rwi-node/spamassessment/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
)

type assessedPage struct {
	bodyPrefix   []byte
	wasBodyWhole bool
	assessment   spamassessment.Assessment
}

func (p assessedPage) headersFrom(upstreamResponseHeaders http.Header) http.Header {
	headers := relayedheaders.EndToEndHeadersOf(upstreamResponseHeaders)
	if p.wasBodyWhole {
		headers.Set("Content-Length", strconv.Itoa(len(p.bodyPrefix)))
	}
	headers.Set(spamassessmenthttpheader.Name, spamassessmenthttpheader.ValueOf(p.assessment))
	return headers
}
