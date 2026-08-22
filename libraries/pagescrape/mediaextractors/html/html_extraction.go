package html

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
)

const (
	mediaHTML  = "text/html"
	mediaXHTML = "application/xhtml+xml"

	msgBaseHrefUnresolved = "base href unresolved, using page url"
	msgPageURLUncanonical = "page url is not canonical, resolving links without it"
)

type HTMLExtraction struct{}

func New() HTMLExtraction {
	return HTMLExtraction{}
}

func (HTMLExtraction) MediaTypes() []string {
	return []string{mediaHTML, mediaXHTML}
}

func (HTMLExtraction) EmittedFormat() contentformatgraph.Format {
	return contentformatgraph.FormatDocumentHTML
}

func (e HTMLExtraction) Extract(
	ctx context.Context,
	pageURL, contentType string,
	body []byte,
) (contentextraction.ExtractedDocument, error) {
	decoded, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		return contentextraction.ExtractedDocument{}, fmt.Errorf("decode charset: %w", err)
	}
	root, err := html.Parse(decoded)
	if err != nil {
		return contentextraction.ExtractedDocument{}, fmt.Errorf("parse html: %w", err)
	}

	scan := scanTree(root)

	var document bytes.Buffer
	if err := html.Render(&document, root); err != nil {
		return contentextraction.ExtractedDocument{}, fmt.Errorf("render html: %w", err)
	}

	links, local, external := discoveredLinks(baseURLOf(ctx, pageURL, scan.baseHref), scan.hrefs)

	return contentextraction.ExtractedDocument{
		Title:                scan.title,
		Body:                 document.Bytes(),
		Format:               e.EmittedFormat(),
		Language:             twoLetterLanguage(scan.language),
		DiscoveredURLs:       links,
		LocalLinks:           local,
		ExternalLinks:        external,
		RefusesIndexing:      scan.noIndex,
		RefusesLinkDiscovery: scan.noFollow,
	}, nil
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

func discoveredLinks(
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

func twoLetterLanguage(language string) string {
	primary := strings.ToLower(strings.TrimSpace(language))
	if dash := strings.IndexByte(primary, '-'); dash >= 0 {
		primary = primary[:dash]
	}
	if len(primary) != 2 {
		return ""
	}
	for _, r := range primary {
		if r < 'a' || r > 'z' {
			return ""
		}
	}
	return primary
}
