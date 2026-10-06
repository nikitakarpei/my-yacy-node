package pageformats

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type formatDerivation interface {
	SourceFormat() documentextraction.Format
	TargetFormat() documentextraction.Format
	BodyFrom(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		source documentextraction.Body,
	) (documentextraction.Body, bool, error)
}
