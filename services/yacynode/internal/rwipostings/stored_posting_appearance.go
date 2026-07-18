package rwipostings

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func appearanceFields(a *yacymodel.Appearance) []*bool {
	return []*bool{
		&a.IndexOf,
		&a.HasLocation,
		&a.HasImage,
		&a.HasAudio,
		&a.HasVideo,
		&a.HasApp,
		&a.AppearsInDescription,
		&a.AppearsInTitle,
		&a.AppearsInCreator,
		&a.AppearsInSubject,
		&a.AppearsInIdentifier,
		&a.Emphasized,
	}
}

func packAppearance(a yacymodel.Appearance) uint16 {
	var bits uint16
	for position, field := range appearanceFields(&a) {
		if *field {
			bits |= 1 << uint(position)
		}
	}
	return bits
}

func unpackAppearance(bits uint16) yacymodel.Appearance {
	var a yacymodel.Appearance
	for position, field := range appearanceFields(&a) {
		*field = bits&(1<<uint(position)) != 0
	}
	return a
}
