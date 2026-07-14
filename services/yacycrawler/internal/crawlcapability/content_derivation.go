package crawlcapability

import "errors"

var ErrUnsupportedSourceFormat = errors.New("unsupported source format")

type ContentDerivation interface {
	Format() PageContentFormat
	Derive(body []byte, sourceFormat PageContentFormat) ([]byte, error)
}
