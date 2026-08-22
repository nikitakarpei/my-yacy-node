package contentformatgraph

import (
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

var ErrUnderivable = errors.New("underivable")

type Derivation interface {
	SourceFormat() documentextraction.Format
	TargetFormat() documentextraction.Format
	Derive(pageURL string, body []byte) ([]byte, error)
}
