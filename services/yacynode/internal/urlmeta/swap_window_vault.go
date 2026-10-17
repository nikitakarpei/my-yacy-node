package urlmeta

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const versionSwapBucket vault.Name = "url_version_swaps"

var errBadSwapWindow = errors.New("bad swap window")

func registerSwapWindows(v *vault.Vault) (*vault.Collection[string, swapWindow], error) {
	windows, err := v.RegisterCollection(
		versionSwapBucket,
		vault.SingleKey(vault.TextKeyPart).KeyLayout(),
		swapWindowCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register swap windows: %w", err)
	}

	return windows, nil
}

type swapWindowCodec struct{}

func (swapWindowCodec) Encode(window swapWindow) ([]byte, error) {
	var stored storedfields.Writer
	stored.Time(window.StartedAt)
	stored.Count(window.Swaps)

	return stored.Record(), nil
}

func (swapWindowCodec) Decode(raw []byte) (swapWindow, error) {
	stored := storedfields.ReaderOf(raw, errBadSwapWindow)
	window := swapWindow{StartedAt: stored.Time("started at"), Swaps: stored.Count("swaps")}
	if err := stored.Err(); err != nil {
		return swapWindow{}, err
	}

	return window, nil
}
