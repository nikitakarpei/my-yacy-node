package yacycrawlcontract

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
)

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
	portHTTP    = "80"
	portHTTPS   = "443"
)

type CanonicalURL struct{ value string }

func (c CanonicalURL) String() string { return c.value }

func (c CanonicalURL) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(c.value)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical url: %w", err)
	}
	return data, nil
}

func (c *CanonicalURL) UnmarshalJSON(data []byte) error {
	var rawURL string
	if err := json.Unmarshal(data, &rawURL); err != nil {
		return fmt.Errorf("unmarshal canonical url: %w", err)
	}
	canonical, err := CanonicalURLOf(rawURL)
	if err != nil {
		return fmt.Errorf("unmarshal canonical url: %w", err)
	}
	if canonical.value != rawURL {
		return fmt.Errorf("unmarshal canonical url: %q is not canonical", rawURL)
	}
	*c = canonical
	return nil
}

func CanonicalURLOf(rawURL string) (CanonicalURL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return CanonicalURL{}, fmt.Errorf("parse url: %w", err)
	}
	return canonicalURLOf(parsed)
}

func CanonicalURLOfReference(baseURL, referenceURL string) (CanonicalURL, error) {
	parsedBaseURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return CanonicalURL{}, fmt.Errorf("parse base url: %w", err)
	}
	parsedReferenceURL, err := url.Parse(strings.TrimSpace(referenceURL))
	if err != nil {
		return CanonicalURL{}, fmt.Errorf("parse reference url: %w", err)
	}
	return canonicalURLOf(parsedBaseURL.ResolveReference(parsedReferenceURL))
}

func canonicalURLOf(parsed *url.URL) (CanonicalURL, error) {
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != schemeHTTP && scheme != schemeHTTPS {
		return CanonicalURL{}, fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return CanonicalURL{}, fmt.Errorf("missing host in %q", parsed.String())
	}
	port := parsed.Port()
	if (scheme == schemeHTTP && port == portHTTP) || (scheme == schemeHTTPS && port == portHTTPS) {
		port = ""
	}

	parsed.Scheme = scheme
	parsed.Host = host
	if port != "" {
		parsed.Host = host + ":" + port
	}
	parsed.Fragment = ""
	parsed.Path = cleanedPathOf(parsed.Path)

	return CanonicalURL{value: parsed.String()}, nil
}

func cleanedPathOf(rawPath string) string {
	if rawPath == "" {
		return "/"
	}
	trailingSlash := strings.HasSuffix(rawPath, "/")
	cleaned := path.Clean(rawPath)
	if cleaned == "." {
		return "/"
	}
	if trailingSlash && !strings.HasSuffix(cleaned, "/") {
		cleaned += "/"
	}
	return cleaned
}
