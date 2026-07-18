package yacymodel

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

func AppearanceFromBitfield(b Bitfield) Appearance {
	return Appearance{
		IndexOf:              b.Get(appearanceFlagBitIndexOf),
		HasLocation:          b.Get(appearanceFlagBitHasLocation),
		HasImage:             b.Get(appearanceFlagBitHasImage),
		HasAudio:             b.Get(appearanceFlagBitHasAudio),
		HasVideo:             b.Get(appearanceFlagBitHasVideo),
		HasApp:               b.Get(appearanceFlagBitHasApp),
		AppearsInDescription: b.Get(appearanceFlagBitAppearsInDescription),
		AppearsInTitle:       b.Get(appearanceFlagBitAppearsInTitle),
		AppearsInCreator:     b.Get(appearanceFlagBitAppearsInCreator),
		AppearsInSubject:     b.Get(appearanceFlagBitAppearsInSubject),
		AppearsInIdentifier:  b.Get(appearanceFlagBitAppearsInIdentifier),
		Emphasized:           b.Get(appearanceFlagBitEmphasized),
	}
}

func (f Appearance) Bitfield() Bitfield {
	b := make(Bitfield, appearanceFlagsByteWidth)
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
