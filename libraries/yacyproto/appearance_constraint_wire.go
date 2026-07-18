package yacyproto

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const appearanceFlagBitCount = appearanceFlagsByteWidth * 8

// appearanceConstraintWireCodec translates between a search request's
// constraint field and the appearance a matching word must have. A peer
// signals "no constraint" with an absent field, an all-zero bitfield, or an
// all-one bitfield; each decodes to no required appearance.
type appearanceConstraintWireCodec struct{}

func (appearanceConstraintWireCodec) decode(
	encoded string,
) (yacymodel.Optional[yacymodel.Appearance], error) {
	none := yacymodel.None[yacymodel.Appearance]()
	if encoded == "" {
		return none, nil
	}
	required, err := decodeBitfield(encoded)
	if err != nil {
		return none, fmt.Errorf("decode appearance constraint: %w", err)
	}
	if required.allSet(appearanceFlagBitCount) {
		return none, nil
	}
	appearance := appearanceFromBitfield(required)
	if appearance == (yacymodel.Appearance{}) {
		return none, nil
	}

	return yacymodel.Some(appearance), nil
}

func (appearanceConstraintWireCodec) encode(
	required yacymodel.Optional[yacymodel.Appearance],
) string {
	appearance, ok := required.Get()
	if !ok {
		return ""
	}

	return yacymodel.Encode(bitfieldFromAppearance(appearance))
}
