package yacymodel

// Appearance describes where and how a word appears in a document.
type Appearance struct {
	IndexOf              bool
	HasLocation          bool
	HasImage             bool
	HasAudio             bool
	HasVideo             bool
	HasApp               bool
	AppearsInDescription bool
	AppearsInTitle       bool
	AppearsInCreator     bool
	AppearsInSubject     bool
	AppearsInIdentifier  bool
	Emphasized           bool
}

// OverlapsAny reports whether the two appearances share at least one trait.
func (a Appearance) OverlapsAny(other Appearance) bool {
	return a.sharedWith(other) != Appearance{}
}

func (a Appearance) sharedWith(other Appearance) Appearance {
	return Appearance{
		IndexOf:              a.IndexOf && other.IndexOf,
		HasLocation:          a.HasLocation && other.HasLocation,
		HasImage:             a.HasImage && other.HasImage,
		HasAudio:             a.HasAudio && other.HasAudio,
		HasVideo:             a.HasVideo && other.HasVideo,
		HasApp:               a.HasApp && other.HasApp,
		AppearsInDescription: a.AppearsInDescription && other.AppearsInDescription,
		AppearsInTitle:       a.AppearsInTitle && other.AppearsInTitle,
		AppearsInCreator:     a.AppearsInCreator && other.AppearsInCreator,
		AppearsInSubject:     a.AppearsInSubject && other.AppearsInSubject,
		AppearsInIdentifier:  a.AppearsInIdentifier && other.AppearsInIdentifier,
		Emphasized:           a.Emphasized && other.Emphasized,
	}
}
