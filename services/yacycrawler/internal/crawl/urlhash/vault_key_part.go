package urlhash

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var errKeyPartLength = errors.New("URL hash key part has the wrong length")

var KeyPart = vault.BytesKeyPartFrom(appendURLHash, urlHashFrom)

func appendURLHash(hash URLHash, buffer []byte) []byte {
	return append(buffer, hash[:]...)
}

func urlHashFrom(stored vault.StoredBytes) (URLHash, error) {
	var hash URLHash
	raw := stored.AppendTo(nil)
	if len(raw) != len(hash) {
		return hash, fmt.Errorf("%w: %d bytes, want %d", errKeyPartLength, len(raw), len(hash))
	}
	copy(hash[:], raw)
	return hash, nil
}
