package rwiescrow

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	heldBucket   vault.Name = "rwi_escrow"
	expiryBucket vault.Name = "rwi_escrow_expiry"
)

const heldPostingBytes = 256

func registerHeldPostings(
	v *vault.Vault,
) (*vault.Collection[postingIdentity, heldPosting], *vault.Set[postingHold], error) {
	held, err := vault.Register(v, heldBucket, heldPostingKeyCodec{}, heldPostingValueCodec{})
	if err != nil {
		return nil, nil, fmt.Errorf("register held postings: %w", err)
	}
	expiry, err := vault.RegisterSet(v, expiryBucket, expiryKeyCodec{})
	if err != nil {
		return nil, nil, fmt.Errorf("register held posting expiry: %w", err)
	}

	return held, expiry, nil
}
