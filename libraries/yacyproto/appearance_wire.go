package yacyproto

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

const appearanceFlagsByteWidth = 4

const (
	appearanceFlagBitIndexOf              = 0
	appearanceFlagBitHasLocation          = 19
	appearanceFlagBitHasImage             = 20
	appearanceFlagBitHasAudio             = 21
	appearanceFlagBitHasVideo             = 22
	appearanceFlagBitHasApp               = 23
	appearanceFlagBitAppearsInDescription = 24
	appearanceFlagBitAppearsInTitle       = 25
	appearanceFlagBitAppearsInCreator     = 26
	appearanceFlagBitAppearsInSubject     = 27
	appearanceFlagBitAppearsInIdentifier  = 28
	appearanceFlagBitEmphasized           = 29
)

func appearanceFromBitfield(b bitfield) yacymodel.Appearance {
	return yacymodel.Appearance{
		IndexOf:              b.get(appearanceFlagBitIndexOf),
		HasLocation:          b.get(appearanceFlagBitHasLocation),
		HasImage:             b.get(appearanceFlagBitHasImage),
		HasAudio:             b.get(appearanceFlagBitHasAudio),
		HasVideo:             b.get(appearanceFlagBitHasVideo),
		HasApp:               b.get(appearanceFlagBitHasApp),
		AppearsInDescription: b.get(appearanceFlagBitAppearsInDescription),
		AppearsInTitle:       b.get(appearanceFlagBitAppearsInTitle),
		AppearsInCreator:     b.get(appearanceFlagBitAppearsInCreator),
		AppearsInSubject:     b.get(appearanceFlagBitAppearsInSubject),
		AppearsInIdentifier:  b.get(appearanceFlagBitAppearsInIdentifier),
		Emphasized:           b.get(appearanceFlagBitEmphasized),
	}
}

func bitfieldFromAppearance(f yacymodel.Appearance) bitfield {
	b := make(bitfield, appearanceFlagsByteWidth)
	b.setBit(appearanceFlagBitIndexOf, f.IndexOf)
	b.setBit(appearanceFlagBitHasLocation, f.HasLocation)
	b.setBit(appearanceFlagBitHasImage, f.HasImage)
	b.setBit(appearanceFlagBitHasAudio, f.HasAudio)
	b.setBit(appearanceFlagBitHasVideo, f.HasVideo)
	b.setBit(appearanceFlagBitHasApp, f.HasApp)
	b.setBit(appearanceFlagBitAppearsInDescription, f.AppearsInDescription)
	b.setBit(appearanceFlagBitAppearsInTitle, f.AppearsInTitle)
	b.setBit(appearanceFlagBitAppearsInCreator, f.AppearsInCreator)
	b.setBit(appearanceFlagBitAppearsInSubject, f.AppearsInSubject)
	b.setBit(appearanceFlagBitAppearsInIdentifier, f.AppearsInIdentifier)
	b.setBit(appearanceFlagBitEmphasized, f.Emphasized)
	return b
}
