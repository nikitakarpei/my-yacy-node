package classificationadding

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/relayedheaders"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
	pageclassificationhttpheader "github.com/nikitakarpei/yacy-rwi-node/pageclassification/httpheader"
)

type classifiedPage struct {
	bodyPrefix     []byte
	wasBodyWhole   bool
	classification pageclassification.Classification
}

func (p classifiedPage) responseFrom(upstreamResponse *http.Response) *http.Response {
	return &http.Response{
		StatusCode: upstreamResponse.StatusCode,
		Header:     p.headersFrom(upstreamResponse.Header),
		Body: pageBody{
			Reader: io.MultiReader(bytes.NewReader(p.bodyPrefix), upstreamResponse.Body),
			Closer: upstreamResponse.Body,
		},
	}
}

func (p classifiedPage) headersFrom(upstreamResponseHeaders http.Header) http.Header {
	headers := relayedheaders.ResponseHeadersFrom(upstreamResponseHeaders)
	if p.wasBodyWhole {
		headers.Set("Content-Length", strconv.Itoa(len(p.bodyPrefix)))
	}
	headers.Set(
		pageclassificationhttpheader.Name,
		pageclassificationhttpheader.ValueOf(p.classification),
	)
	return headers
}

type pageBody struct {
	io.Reader
	io.Closer
}
