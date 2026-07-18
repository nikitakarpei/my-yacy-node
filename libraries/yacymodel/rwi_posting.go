package yacymodel

// RWIPosting is one word's appearance in one document: a reverse-word-index
// entry.
type RWIPosting struct {
	WordHash               Hash
	URLHash                URLHash
	LastModified           MicroDate
	TitleWords             uint8
	TextWords              uint16
	Phrases                uint16
	DocumentType           DocumentType
	Language               Language
	LocalLinks             uint8
	ExternalLinks          uint8
	URLLength              uint8
	URLComponents          uint8
	Appearance             Appearance
	Hits                   uint8
	TextPosition           uint16
	PhraseRelativePosition uint8
	PhrasePosition         uint8
}
