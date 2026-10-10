// Package tlsclient fetches pages through the egress proxy with the TLS and
// HTTP/2 fingerprint of Chrome, and decodes their bodies.
package tlsclient

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/chromenavigation"
)

const contentEncodingName = "Content-Encoding"

var decodableContentEncodings = map[string]string{
	"gzip":    "gzip",
	"x-gzip":  "gzip",
	"deflate": "deflate",
	"br":      "br",
	"zstd":    "zstd",
}

type PageFetcher struct {
	client    tls_client.HttpClient
	observers Observers
}

func New(
	egressProxyURL *url.URL,
	trustedRoots *x509.CertPool,
	fetchTimeout time.Duration,
	observers Observers,
) (*PageFetcher, error) {
	profileName := fmt.Sprintf("chrome_%d", chromenavigation.MajorVersion)
	profile, found := profiles.MappedTLSClients[profileName]
	if !found {
		return nil, fmt.Errorf("tls-client has no profile %s", profileName)
	}
	client, err := tls_client.NewHttpClient(nil,
		tls_client.WithClientProfile(profile),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithProxyUrl(egressProxyURL.String()),
		tls_client.WithTimeoutMilliseconds(int(fetchTimeout.Milliseconds())),
		tls_client.WithTransportOptions(&tls_client.TransportOptions{
			RootCAs:            trustedRoots,
			DisableCompression: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("build tls client: %w", err)
	}
	return &PageFetcher{client: client, observers: observers}, nil
}

func (f *PageFetcher) Fetch(
	ctx context.Context,
	request chromenavigation.Request,
) (*http.Response, bool) {
	address := request.Address.String()
	response, err := f.client.Do(vendorRequestFor(ctx, request))
	if err != nil {
		f.reportFailure(ctx, address, err)
		return nil, false
	}
	f.observers.PageFetched(ctx, address, response.StatusCode)
	return decodedPageFrom(response), true
}

func vendorRequestFor(ctx context.Context, request chromenavigation.Request) *fhttp.Request {
	headers := fhttp.Header{}
	headerOrder := make([]string, 0, len(request.HeaderFields))
	for _, headerField := range request.HeaderFields {
		headers[headerField.Name] = []string{headerField.Value}
		headerOrder = append(headerOrder, strings.ToLower(headerField.Name))
	}
	headers[fhttp.HeaderOrderKey] = headerOrder
	return (&fhttp.Request{
		Method:     request.Method,
		URL:        request.Address,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     headers,
		Host:       request.Address.Host,
	}).WithContext(ctx)
}

func (f *PageFetcher) reportFailure(ctx context.Context, address string, cause error) {
	if ctx.Err() != nil {
		f.observers.FetchCancelled(ctx, address)
		return
	}
	f.observers.FetchFailed(ctx, address, cause)
}

func decodedPageFrom(response *fhttp.Response) *http.Response {
	page := &http.Response{
		Status:        response.Status,
		StatusCode:    response.StatusCode,
		Proto:         response.Proto,
		ProtoMajor:    response.ProtoMajor,
		ProtoMinor:    response.ProtoMinor,
		Header:        http.Header(response.Header),
		Body:          response.Body,
		ContentLength: response.ContentLength,
	}
	contentEncodings := page.Header.Values(contentEncodingName)
	if len(contentEncodings) != 1 {
		return page
	}
	decoding, found := decodableContentEncodings[strings.ToLower(
		strings.TrimSpace(contentEncodings[0]),
	)]
	if !found {
		return page
	}
	page.Header.Del(contentEncodingName)
	page.Header.Del("Content-Length")
	page.ContentLength = -1
	if hasContent(response) {
		page.Body = fhttp.DecompressBodyByType(response.Body, decoding)
	}
	return page
}

func hasContent(response *fhttp.Response) bool {
	return response.Request.Method != http.MethodHead &&
		response.StatusCode != http.StatusNoContent &&
		response.StatusCode != http.StatusNotModified
}
