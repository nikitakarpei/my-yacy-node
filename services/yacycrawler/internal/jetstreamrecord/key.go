package jetstreamrecord

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func KeyOf(parts ...string) string {
	digests := make([]string, 0, len(parts))
	for _, part := range parts {
		sum := sha256.Sum256([]byte(part))
		digests = append(digests, hex.EncodeToString(sum[:]))
	}
	return strings.Join(digests, ".")
}
