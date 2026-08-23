// Package pagelinks reads the links an HTML page points to and the robots
// directives its markup states about indexing them and following them.
package pagelinks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	mediaHTML  = "text/html"
	mediaXHTML = "application/xhtml+xml"

	msgMediaTypeUnparsed  = "content type unparsed, falling back to its leading segment"
	msgBaseHrefUnresolved = "base href unresolved, using page url"
	msgPageURLUncanonical = "page url is not canonical, resolving links without it"
)

var ErrNotHTML = errors.New("not an html page")

type PageLinks struct {
	LinkedURLs           []canonicalurl.CanonicalURL
	LocalLinks           int
	ExternalLinks        int
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}

func PageLinksFrom(
	ctx context.Context,
	pageURL, contentType string,
	body []byte,
) (PageLinks, error) {
	if !isHTML(ctx, contentType) {
		return PageLinks{}, ErrNotHTML
	}
	decoded, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		return PageLinks{}, fmt.Errorf("decode charset: %w", err)
	}
	root, err := html.Parse(decoded)
	if err != nil {
		return PageLinks{}, fmt.Errorf("parse html: %w", err)
	}
	return PageLinksOf(ctx, pageURL, root), nil
}

func PageLinksOf(ctx context.Context, pageURL string, root *html.Node) PageLinks {
	scan := scanLinkTree(root)
	links := PageLinks{
		RefusesIndexing:      scan.noIndex,
		RefusesLinkDiscovery: scan.noFollow,
	}
	links.LinkedURLs, links.LocalLinks, links.ExternalLinks = linkedURLs(
		baseURLOf(ctx, pageURL, scan.baseHref), scan.hrefs,
	)
	return links
}

func isHTML(ctx context.Context, contentType string) bool {
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		slog.DebugContext(ctx, msgMediaTypeUnparsed,
			slog.String("contentType", contentType),
			slog.Any("error", err),
		)
		media = strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	}
	return media == mediaHTML || media == mediaXHTML
}

func baseURLOf(ctx context.Context, pageURL, baseHref string) canonicalurl.CanonicalURL {
	canonicalPageURL, err := canonicalurl.CanonicalURLOf(pageURL)
	if err != nil {
		slog.DebugContext(ctx, msgPageURLUncanonical,
			slog.String("url", pageURL),
			slog.Any("error", err),
		)
		return canonicalurl.CanonicalURL{}
	}
	if baseHref == "" {
		return canonicalPageURL
	}
	base, err := canonicalPageURL.CanonicalURLOfLink(baseHref)
	if err != nil {
		slog.DebugContext(ctx, msgBaseHrefUnresolved,
			slog.String("url", pageURL),
			slog.String("baseHref", baseHref),
			slog.Any("error", err),
		)
		return canonicalPageURL
	}
	return base
}

func linkedURLs(
	baseURL canonicalurl.CanonicalURL,
	hrefs []string,
) (links []canonicalurl.CanonicalURL, local, external int) {
	baseHost := baseURL.Hostname()
	seen := map[canonicalurl.CanonicalURL]struct{}{}
	for _, href := range hrefs {
		canonical, err := baseURL.CanonicalURLOfLink(href)
		if err != nil {
			continue
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		links = append(links, canonical)
		if canonical.Hostname() == baseHost {
			local++
		} else {
			external++
		}
	}
	return links, local, external
}
