package yacyproto

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func hashOfTheAddress(address string, sentHash string) (yacymodel.URLHash, error) {
	hashTheSenderNames, err := yacymodel.ParseURLHash(sentHash)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("url metadata hash: %w", err)
	}
	hashOfTheAddress, err := yacymodel.URLHashOf(address)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("url metadata hash: %w", err)
	}
	if hashTheSenderNames != hashOfTheAddress {
		return yacymodel.URLHash{}, fmt.Errorf(
			"url metadata hash: the sender names %s for the address %q, which holds the hash %s",
			hashTheSenderNames, address, hashOfTheAddress,
		)
	}

	return hashOfTheAddress, nil
}
