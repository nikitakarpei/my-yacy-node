package contentformatgraph

import "errors"

var ErrUnextractable = errors.New("unextractable")

type Derivation interface {
	SourceFormat() Format
	TargetFormat() Format
	Derive(pageURL string, body []byte) ([]byte, error)
}
