// Package markdown is the markdown page representation recall yields for a URL.
package markdown

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
)

const Kind recall.RepresentationKind = "markdown"

type Page struct {
	CanonicalURL canonicalurl.CanonicalURL
	Markdown     string
}
