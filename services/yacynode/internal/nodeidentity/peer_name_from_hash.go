package nodeidentity

import (
	"crypto/md5"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const derivedPeerNameSerialRange = 10000

func PeerNameFromHash(hash yacymodel.Hash) (yacymodel.PeerName, error) {
	spread := md5.Sum([]byte(hash.String()))
	text := fmt.Sprintf(
		"%s-%s-%04d",
		peerNameAdjectives[int(spread[0])%len(peerNameAdjectives)],
		peerNameNouns[int(spread[1])%len(peerNameNouns)],
		(int(spread[2])<<8|int(spread[3]))%derivedPeerNameSerialRange,
	)

	name, err := yacymodel.ParsePeerName(text)
	if err != nil {
		return yacymodel.PeerName{}, fmt.Errorf("derive peer name from %s: %w", hash, err)
	}

	return name, nil
}
