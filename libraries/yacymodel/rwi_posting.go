package yacymodel

// RWIPosting is one word's appearance in one document: a reverse-word-index
// entry.
type RWIPosting struct {
	WordHash               Hash
	URLHash                URLHash
	LastModified           MicroDate
	TitleWordCount         uint8
	TextWordCount          uint16
	PhraseCount            uint16
	DocType                byte
	Language               Language
	LocalLinkCount         uint8
	ExternalLinkCount      uint8
	URLLength              uint8
	URLComponentCount      uint8
	Flags                  AppearanceFlags
	HitCount               uint8
	TextPosition           uint16
	PhraseRelativePosition uint8
	PhrasePosition         uint8
}
