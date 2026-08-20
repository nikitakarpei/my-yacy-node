package yacycrawlcontract

import (
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

func CanonicalURLOf(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	return canonicalURLOf(parsed)
}

func CanonicalURLOfReference(baseURL, referenceURL string) (string, error) {
	parsedBaseURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}
	parsedReferenceURL, err := url.Parse(strings.TrimSpace(referenceURL))
	if err != nil {
		return "", fmt.Errorf("parse reference url: %w", err)
	}
	return canonicalURLOf(parsedBaseURL.ResolveReference(parsedReferenceURL))
}

func canonicalURLOf(parsed *url.URL) (string, error) {
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != schemeHTTP && scheme != schemeHTTPS {
		return "", fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("missing host in %q", parsed.String())
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

	return parsed.String(), nil
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
