package pageformats

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type FormatDerivation interface {
	SourceFormat() documentextraction.Format
	TargetFormat() documentextraction.Format
	BodyFrom(pageURL canonicalurl.CanonicalURL, sourceBody []byte) ([]byte, bool, error)
}
