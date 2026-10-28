package pagereading

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
)

type BodyReading struct {
	Refusals   robotsmeta.Refusals
	LinkedURLs []canonicalurl.CanonicalURL
}
