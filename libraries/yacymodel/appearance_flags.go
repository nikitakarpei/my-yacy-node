package yacymodel

// AppearanceFlags describes where and how a word appears in a document.
type AppearanceFlags struct {
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
