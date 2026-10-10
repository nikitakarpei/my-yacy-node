package chromenavigation_test

import (
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/chromenavigation"
)

const chromeUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36"

func TestASecurePageIsAskedWithTheHeadersOfChromeInTheirOrder(t *testing.T) {
	request := chromenavigation.New("").RequestFor(
		http.MethodGet, addressOf(t, "https://site.example/page"), http.Header{
			"User-Agent":       {"yacy"},
			"Accept-Encoding":  {"identity"},
			"Proxy-Connection": {"keep-alive"},
		},
	)

	want := []chromenavigation.HeaderField{
		{
			Name:  "sec-ch-ua",
			Value: `"Chromium";v="152", "Not?A_Brand";v="24", "Google Chrome";v="152"`,
		},
		{Name: "sec-ch-ua-mobile", Value: "?0"},
		{Name: "sec-ch-ua-platform", Value: `"Windows"`},
		{Name: "upgrade-insecure-requests", Value: "1"},
		{Name: "user-agent", Value: chromeUserAgent},
		{Name: "accept", Value: "text/html,application/xhtml+xml,application/xml;q=0.9," +
			"image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		{Name: "sec-fetch-site", Value: "none"},
		{Name: "sec-fetch-mode", Value: "navigate"},
		{Name: "sec-fetch-user", Value: "?1"},
		{Name: "sec-fetch-dest", Value: "document"},
		{Name: "accept-encoding", Value: "gzip, deflate, br, zstd"},
		{Name: "accept-language", Value: "en-US,en;q=0.9"},
		{Name: "priority", Value: "u=0, i"},
	}
	if request.Method != http.MethodGet ||
		request.Address.String() != "https://site.example/page" ||
		!slices.Equal(request.HeaderFields, want) {
		t.Fatalf("request = %+v", request)
	}
}

func TestAnInsecurePageIsAskedWithoutTheHeadersChromeKeepsForSecurePages(t *testing.T) {
	request := chromenavigation.New("").RequestFor(
		http.MethodHead, addressOf(t, "http://site.example/page"), http.Header{},
	)

	want := []string{
		"Host", "Connection", "Upgrade-Insecure-Requests", "User-Agent", "Accept",
		"Accept-Encoding", "Accept-Language",
	}
	if !slices.Equal(namesOf(request.HeaderFields), want) ||
		valueOf(request.HeaderFields, "Accept-Encoding") != "gzip, deflate" {
		t.Fatalf("header fields = %+v", request.HeaderFields)
	}
}

func TestTheCookieRefererAndValidatorsOfTheClientAreKept(t *testing.T) {
	clientHeaders := http.Header{
		"Cookie":            {"session=1"},
		"Referer":           {"https://site.example/"},
		"If-None-Match":     {`"v1"`},
		"If-Modified-Since": {"Mon, 05 Oct 2026 10:00:00 GMT"},
	}

	request := chromenavigation.New("https://www.google.com/").RequestFor(
		http.MethodGet, addressOf(t, "https://site.example/page"), clientHeaders,
	)

	if valueOf(request.HeaderFields, "cookie") != "session=1" ||
		valueOf(request.HeaderFields, "referer") != "https://site.example/" ||
		valueOf(request.HeaderFields, "sec-fetch-site") != "same-origin" ||
		valueOf(request.HeaderFields, "if-none-match") != `"v1"` ||
		valueOf(request.HeaderFields, "if-modified-since") != "Mon, 05 Oct 2026 10:00:00 GMT" {
		t.Fatalf("header fields = %+v", request.HeaderFields)
	}
	names := namesOf(request.HeaderFields)
	if slices.Index(names, "sec-fetch-dest") > slices.Index(names, "referer") ||
		slices.Index(names, "accept-language") > slices.Index(names, "cookie") ||
		slices.Index(names, "if-modified-since") > slices.Index(names, "priority") {
		t.Fatalf("header names = %v", names)
	}
}

func TestTheDefaultRefererIsSentWhenTheClientSendsNone(t *testing.T) {
	navigator := chromenavigation.New("https://www.google.com/")

	secureRequest := navigator.RequestFor(
		http.MethodGet, addressOf(t, "https://site.example/page"), http.Header{},
	)
	insecureRequest := navigator.RequestFor(
		http.MethodGet, addressOf(t, "http://site.example/page"), http.Header{},
	)

	if valueOf(secureRequest.HeaderFields, "referer") != "https://www.google.com/" ||
		valueOf(secureRequest.HeaderFields, "sec-fetch-site") != "cross-site" ||
		valueOf(insecureRequest.HeaderFields, "Referer") != "https://www.google.com/" {
		t.Fatalf(
			"secure %+v, insecure %+v",
			secureRequest.HeaderFields,
			insecureRequest.HeaderFields,
		)
	}
}

func addressOf(t *testing.T, rawAddress string) *url.URL {
	t.Helper()
	address, err := url.Parse(rawAddress)
	if err != nil {
		t.Fatal(err)
	}
	return address
}

func namesOf(headerFields []chromenavigation.HeaderField) []string {
	names := make([]string, 0, len(headerFields))
	for _, headerField := range headerFields {
		names = append(names, headerField.Name)
	}
	return names
}

func valueOf(headerFields []chromenavigation.HeaderField, name string) string {
	for _, headerField := range headerFields {
		if headerField.Name == name {
			return headerField.Value
		}
	}
	return ""
}
