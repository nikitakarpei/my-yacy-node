package pageformats

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type Derivation interface {
	SourceFormat() documentextraction.Format
	TargetFormat() documentextraction.Format
	Derive(pageURL canonicalurl.CanonicalURL, body []byte) ([]byte, bool, error)
}
