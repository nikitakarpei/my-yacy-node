package searchpages

import (
	"encoding/xml"
	"net/http"
)

const (
	openSearchContentType = "application/opensearchdescription+xml"
	searchTermsTemplate   = "{searchTerms}"
	forwardedProtoHeader  = "X-Forwarded-Proto"
	secureScheme          = "https"
	plainScheme           = "http"
)

type openSearchDescriptionWireForm struct {
	XMLName       xml.Name              `xml:"http://a9.com/-/spec/opensearch/1.1/ OpenSearchDescription"`
	ShortName     string                `xml:"ShortName"`
	InputEncoding string                `xml:"InputEncoding"`
	URL           openSearchURLWireForm `xml:"Url"`
}

type openSearchURLWireForm struct {
	Type     string `xml:"type,attr"`
	Method   string `xml:"method,attr"`
	Template string `xml:"template,attr"`
}

func showOpenSearchDescription(w http.ResponseWriter, r *http.Request) {
	description := openSearchDescriptionWireForm{
		ShortName:     r.Host,
		InputEncoding: "UTF-8",
		URL: openSearchURLWireForm{
			Type:   "text/html",
			Method: http.MethodGet,
			Template: schemeOf(r) + "://" + r.Host + searchPath + "?" +
				queryParam + "=" + searchTermsTemplate,
		},
	}
	w.Header().Set("Content-Type", openSearchContentType)
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(description)
}

func schemeOf(r *http.Request) string {
	if r.TLS != nil || r.Header.Get(forwardedProtoHeader) == secureScheme {
		return secureScheme
	}
	return plainScheme
}
