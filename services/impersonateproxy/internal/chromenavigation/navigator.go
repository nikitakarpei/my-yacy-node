// Package chromenavigation holds the request that Chrome sends when it opens a
// page, with the header fields of Chrome in the order of Chrome.
package chromenavigation

import (
	"fmt"
	"net/http"
	"net/url"
)

const MajorVersion = 152

const (
	acceptedMediaTypes = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif," +
		"image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"
	acceptedLanguages = "en-US,en;q=0.9"
	refererName       = "Referer"
)

var userAgent = fmt.Sprintf(
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "+
		"(KHTML, like Gecko) Chrome/%d.0.0.0 Safari/537.36",
	MajorVersion,
)

type HeaderField struct {
	Name  string
	Value string
}

type Request struct {
	Method       string
	Address      *url.URL
	HeaderFields []HeaderField
}

type Navigator struct {
	defaultReferer string
}

func New(defaultReferer string) *Navigator {
	return &Navigator{defaultReferer: defaultReferer}
}

func (n *Navigator) RequestFor(method string, address *url.URL, clientHeaders http.Header) Request {
	referer := n.refererFrom(clientHeaders)
	headerFields := insecureHeaderFieldsFor(address, referer, clientHeaders)
	if address.Scheme == "https" {
		headerFields = secureHeaderFieldsFor(address, referer, clientHeaders)
	}
	return Request{Method: method, Address: address, HeaderFields: headerFields}
}

func (n *Navigator) refererFrom(clientHeaders http.Header) string {
	if referer := clientHeaders.Get(refererName); referer != "" {
		return referer
	}
	return n.defaultReferer
}

func insecureHeaderFieldsFor(
	address *url.URL,
	referer string,
	clientHeaders http.Header,
) []HeaderField {
	return presentFieldsOf([]HeaderField{
		{Name: "Host", Value: address.Host},
		{Name: "Connection", Value: "keep-alive"},
		{Name: "Upgrade-Insecure-Requests", Value: "1"},
		{Name: "User-Agent", Value: userAgent},
		{Name: "Accept", Value: acceptedMediaTypes},
		{Name: refererName, Value: referer},
		{Name: "Accept-Encoding", Value: "gzip, deflate"},
		{Name: "Accept-Language", Value: acceptedLanguages},
		{Name: "Cookie", Value: clientHeaders.Get("Cookie")},
		{Name: "If-None-Match", Value: clientHeaders.Get("If-None-Match")},
		{Name: "If-Modified-Since", Value: clientHeaders.Get("If-Modified-Since")},
	})
}

func presentFieldsOf(headerFields []HeaderField) []HeaderField {
	presentFields := make([]HeaderField, 0, len(headerFields))
	for _, headerField := range headerFields {
		if headerField.Value != "" {
			presentFields = append(presentFields, headerField)
		}
	}
	return presentFields
}

func secureHeaderFieldsFor(
	address *url.URL,
	referer string,
	clientHeaders http.Header,
) []HeaderField {
	return presentFieldsOf([]HeaderField{
		{Name: "sec-ch-ua", Value: brandList()},
		{Name: "sec-ch-ua-mobile", Value: "?0"},
		{Name: "sec-ch-ua-platform", Value: `"Windows"`},
		{Name: "upgrade-insecure-requests", Value: "1"},
		{Name: "user-agent", Value: userAgent},
		{Name: "accept", Value: acceptedMediaTypes},
		{Name: "sec-fetch-site", Value: fetchSiteFor(address, referer)},
		{Name: "sec-fetch-mode", Value: "navigate"},
		{Name: "sec-fetch-user", Value: "?1"},
		{Name: "sec-fetch-dest", Value: "document"},
		{Name: "referer", Value: referer},
		{Name: "accept-encoding", Value: "gzip, deflate, br, zstd"},
		{Name: "accept-language", Value: acceptedLanguages},
		{Name: "cookie", Value: clientHeaders.Get("Cookie")},
		{Name: "if-none-match", Value: clientHeaders.Get("If-None-Match")},
		{Name: "if-modified-since", Value: clientHeaders.Get("If-Modified-Since")},
		{Name: "priority", Value: "u=0, i"},
	})
}

func brandList() string {
	greaseCharacters := []string{" ", "(", ":", "-", ".", "/", ")", ";", "=", "?", "_"}
	greaseVersions := []string{"8", "99", "24"}
	brandOrders := [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	greaseBrand := fmt.Sprintf(`"Not%sA%sBrand";v="%s"`,
		greaseCharacters[MajorVersion%len(greaseCharacters)],
		greaseCharacters[(MajorVersion+1)%len(greaseCharacters)],
		greaseVersions[MajorVersion%len(greaseVersions)],
	)
	brandOrder := brandOrders[MajorVersion%len(brandOrders)]
	var brands [3]string
	brands[brandOrder[0]] = greaseBrand
	brands[brandOrder[1]] = fmt.Sprintf(`"Chromium";v="%d"`, MajorVersion)
	brands[brandOrder[2]] = fmt.Sprintf(`"Google Chrome";v="%d"`, MajorVersion)
	return brands[0] + ", " + brands[1] + ", " + brands[2]
}

func fetchSiteFor(address *url.URL, referer string) string {
	if referer == "" {
		return "none"
	}
	refererAddress, err := url.Parse(referer)
	if err == nil && refererAddress.Scheme == address.Scheme &&
		refererAddress.Host == address.Host {
		return "same-origin"
	}
	return "cross-site"
}
