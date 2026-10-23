package markdownrecall

import (
	"crypto/sha256"
	"encoding/base64"
	"time"
)

const versionPrefix = "SHA-256="

type StoredMarkdown struct {
	Markdown []byte
	StoredAt time.Time
	Version  string
}

func VersionOf(markdown []byte) string {
	sum := sha256.Sum256(markdown)
	return versionPrefix + base64.URLEncoding.EncodeToString(sum[:])
}
