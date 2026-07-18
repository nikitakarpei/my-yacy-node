package yacymodel

// DocumentType is the media kind of the document an RWI posting indexes.
type DocumentType uint8

const (
	DocumentTypeUnknown DocumentType = iota
	DocumentTypeText
	DocumentTypeHTML
	DocumentTypeDocument
	DocumentTypeImage
	DocumentTypeMovie
	DocumentTypeFlash
	DocumentTypeShare
	DocumentTypeAudio
	DocumentTypePDF
	DocumentTypeBinary
)
