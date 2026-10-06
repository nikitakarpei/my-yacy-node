package documentextraction

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction/bodies/htmltree"
)

const (
	mediaHTML  = "text/html"
	mediaXHTML = "application/xhtml+xml"
)

type htmlExtraction struct{}

func newHTMLExtraction() htmlExtraction {
	return htmlExtraction{}
}

func (htmlExtraction) MediaTypes() []string {
	return []string{mediaHTML, mediaXHTML}
}

func (htmlExtraction) EmittedFormat() Format {
	return FormatDocumentHTML
}

func (htmlExtraction) DocumentFrom(
	ctx context.Context,
	fetchedBody []byte,
	contentType string,
	pageURL canonicalurl.CanonicalURL,
) (Document, error) {
	decoded, err := charset.NewReader(bytes.NewReader(fetchedBody), contentType)
	if err != nil {
		return Document{}, fmt.Errorf("decode charset: %w", err)
	}
	root, err := html.Parse(decoded)
	if err != nil {
		return Document{}, fmt.Errorf("parse html: %w", err)
	}

	scan := scanTree(root)
	baseURL := baseURLOf(ctx, pageURL, scan.baseHref)
	links := distinctLinksFrom(scan.hrefs, baseURL)
	baseHost := baseURL.Hostname()

	return Document{
		Title:         scan.title,
		Body:          htmltree.New(root),
		Format:        FormatDocumentHTML,
		Language:      twoLetterLanguage(scan.language),
		LocalLinks:    localLinksOf(links, baseHost),
		ExternalLinks: externalLinksOf(links, baseHost),
	}, nil
}

// TECHDEBT: Naming — derivation: twoLetterLanguage returns a value derived from a language tag but carries no preposition to its source
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
