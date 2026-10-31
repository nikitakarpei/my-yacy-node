package pagefingerprint

import (
	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
)

func Write(stored *storedfields.Writer, fingerprint Fingerprint) {
	stored.Varint(int64(fingerprint.Modified))
	stored.Count(int(fingerprint.LocalLinks))
	stored.Count(int(fingerprint.ExternalLinks))
	stored.Count(int(fingerprint.TitleWords))
	stored.Text(fingerprint.Language.String())
	stored.Byte(byte(fingerprint.DocumentType))
}
