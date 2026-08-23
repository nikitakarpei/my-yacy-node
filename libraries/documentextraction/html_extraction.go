package documentextraction

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
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
	body []byte,
	contentType string,
	pageURL canonicalurl.CanonicalURL,
) (Document, error) {
	decoded, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		return Document{}, fmt.Errorf("decode charset: %w", err)
	}
	root, err := html.Parse(decoded)
	if err != nil {
		return Document{}, fmt.Errorf("parse html: %w", err)
	}

	scan := scanTree(root)

	var document bytes.Buffer
	if err := html.Render(&document, root); err != nil {
		return Document{}, fmt.Errorf("render html: %w", err)
	}

	baseURL := baseURLOf(ctx, pageURL, scan.baseHref)
	links := distinctLinksFrom(scan.hrefs, baseURL)
	baseHost := baseURL.Hostname()

	return Document{
		Title:         scan.title,
		Body:          document.Bytes(),
		Format:        FormatDocumentHTML,
		Language:      twoLetterLanguage(scan.language),
		LocalLinks:    localLinksOf(links, baseHost),
		ExternalLinks: externalLinksOf(links, baseHost),
	}, nil
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
