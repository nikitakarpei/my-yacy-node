package pageversion

import (
	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
)

func AppendBytes(version Version, buffer []byte) []byte {
	var stored storedfields.Writer
	pagefingerprint.Write(&stored, version.fingerprint)
	if root, rooted := version.root.Get(); rooted {
		stored.Bytes(root.Bytes())
	}

	return append(buffer, stored.Record()...)
}
