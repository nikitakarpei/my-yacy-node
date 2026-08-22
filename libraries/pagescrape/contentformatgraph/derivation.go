package contentformatgraph

import "errors"

var ErrUnderivable = errors.New("underivable")

type Derivation interface {
	SourceFormat() Format
	TargetFormat() Format
	Derive(pageURL string, body []byte) ([]byte, error)
}
