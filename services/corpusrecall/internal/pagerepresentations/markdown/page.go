// Package markdown is the markdown page representation recall yields for a URL.
package markdown

import (
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const Kind recall.RepresentationKind = "markdown"

type Page struct {
	CanonicalURL yacycrawlcontract.CanonicalURL
	Markdown     string
}
