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

type CanonicalURL struct {
	value    string
	hostname string
	hasQuery bool
}

func (c CanonicalURL) String() string { return c.value }

func (c CanonicalURL) Hostname() string { return c.hostname }

func (c CanonicalURL) HasQuery() bool { return c.hasQuery }

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

func CanonicalURLOfLink(linkURL, baseURL string) (CanonicalURL, error) {
	parsedLinkURL, err := url.Parse(strings.TrimSpace(linkURL))
	if err != nil {
		return CanonicalURL{}, fmt.Errorf("parse link url: %w", err)
	}
	parsedBaseURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return CanonicalURL{}, fmt.Errorf("parse base url: %w", err)
	}
	return canonicalURLOf(parsedBaseURL.ResolveReference(parsedLinkURL))
}

func canonicalURLOf(parsed *url.URL) (CanonicalURL, error) {
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != schemeHTTP && scheme != schemeHTTPS {
		return CanonicalURL{}, fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return CanonicalURL{}, fmt.Errorf("missing host in %q", parsed.String())
	}
	port := parsed.Port()
	if (scheme == schemeHTTP && port == portHTTP) || (scheme == schemeHTTPS && port == portHTTPS) {
		port = ""
	}

	parsed.Scheme = scheme
	parsed.Host = hostname
	if port != "" {
		parsed.Host = hostname + ":" + port
	}
	parsed.Fragment = ""
	parsed.Path = cleanedPathOf(parsed.Path)

	return CanonicalURL{
		value:    parsed.String(),
		hostname: hostname,
		hasQuery: parsed.RawQuery != "",
	}, nil
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
