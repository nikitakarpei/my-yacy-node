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
