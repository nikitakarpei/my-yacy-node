package html

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

const (
	mediaHTML  = "text/html"
	mediaXHTML = "application/xhtml+xml"

	msgBaseHrefUnresolved = "base href unresolved, using page url"
)

type HTMLExtraction struct{}

func New() HTMLExtraction {
	return HTMLExtraction{}
}

func (HTMLExtraction) MediaTypes() []string {
	return []string{mediaHTML, mediaXHTML}
}

func (HTMLExtraction) Extract(
	ctx context.Context,
	pageURL, contentType string,
	body []byte,
) (contentextraction.ExtractedContent, error) {
	decoded, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		return contentextraction.ExtractedContent{}, fmt.Errorf("decode charset: %w", err)
	}
	root, err := html.Parse(decoded)
	if err != nil {
		return contentextraction.ExtractedContent{}, fmt.Errorf("parse html: %w", err)
	}

	scan := scanTree(root)

	var document bytes.Buffer
	if err := html.Render(&document, root); err != nil {
		return contentextraction.ExtractedContent{}, fmt.Errorf("render html: %w", err)
	}

	base := pageURL
	if scan.baseHref != "" {
		if resolved, resolveErr := resolveBase(pageURL, scan.baseHref); resolveErr == nil {
			base = resolved
		} else {
			slog.DebugContext(ctx, msgBaseHrefUnresolved,
				slog.String("url", pageURL),
				slog.String("baseHref", scan.baseHref),
				slog.Any("error", resolveErr),
			)
		}
	}
	links, local, external := resolveLinks(base, scan.hrefs)

	return contentextraction.ExtractedContent{
		Title:                scan.title,
		Body:                 document.Bytes(),
		Format:               contentformatgraph.FormatDocumentHTML,
		Language:             twoLetterLanguage(scan.language),
		DiscoveredURLs:       links,
		LocalLinks:           local,
		ExternalLinks:        external,
		RefusesIndexing:      scan.noIndex,
		RefusesLinkDiscovery: scan.noFollow,
	}, nil
}

func resolveBase(pageURL, baseHref string) (string, error) {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("parse page url: %w", err)
	}
	ref, err := url.Parse(baseHref)
	if err != nil {
		return "", fmt.Errorf("parse base href: %w", err)
	}
	return parsed.ResolveReference(ref).String(), nil
}

func resolveLinks(base string, hrefs []string) (links []string, local, external int) {
	baseHost := hostOf(base)
	seen := map[string]struct{}{}
	for _, href := range hrefs {
		canonical, err := canonicalurl.ResolveReference(base, href)
		if err != nil {
			continue
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		links = append(links, canonical)
		if hostOf(canonical) == baseHost {
			local++
		} else {
			external++
		}
	}
	return links, local, external
}

func hostOf(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
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
