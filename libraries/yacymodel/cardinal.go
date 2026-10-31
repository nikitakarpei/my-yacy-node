package yacymodel

func WrappedCardinalFrom[Cardinal int | uint32](cardinal Cardinal) uint8 {
	return uint8(cardinal)
}

func SaturatedCardinalFrom[Cardinal uint16 | uint32](cardinal int) Cardinal {
	ceiling := ^Cardinal(0)
	if uint64(cardinal) > uint64(ceiling) { //nolint:gosec // a cardinal is never negative.
		return ceiling
	}

	return Cardinal(cardinal)
}
