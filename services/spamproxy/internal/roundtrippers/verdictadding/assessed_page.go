package verdictadding

import (
	"bytes"
	"io"
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

func (p assessedPage) responseFrom(upstreamResponse *http.Response) *http.Response {
	return &http.Response{
		StatusCode: upstreamResponse.StatusCode,
		Header:     p.headersFrom(upstreamResponse.Header),
		Body: pageBody{
			Reader: io.MultiReader(bytes.NewReader(p.bodyPrefix), upstreamResponse.Body),
			Closer: upstreamResponse.Body,
		},
	}
}

func (p assessedPage) headersFrom(upstreamResponseHeaders http.Header) http.Header {
	headers := relayedheaders.ResponseHeadersFrom(upstreamResponseHeaders)
	if p.wasBodyWhole {
		headers.Set("Content-Length", strconv.Itoa(len(p.bodyPrefix)))
	}
	headers.Set(spamassessmenthttpheader.Name, spamassessmenthttpheader.ValueOf(p.assessment))
	return headers
}

type pageBody struct {
	io.Reader
	io.Closer
}
