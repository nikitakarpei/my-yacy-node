package readabletext

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type FullTextDerivation struct{}

func FromFullText() FullTextDerivation {
	return FullTextDerivation{}
}

func (FullTextDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatFullText
}

func (FullTextDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatReadableText
}

func (FullTextDerivation) BodyFrom(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	body []byte,
) ([]byte, bool, error) {
	return body, true, nil
}
