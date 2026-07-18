package yacymodel

// RWIPosting is one word's appearance in one document: a reverse-word-index
// entry.
type RWIPosting struct {
	WordHash               Hash
	URLHash                URLHash
	LastModified           MicroDate
	TitleWords             int
	TextWords              int
	Phrases                int
	DocumentType           DocumentType
	Language               Language
	LocalLinks             int
	ExternalLinks          int
	URLLength              int
	URLComponents          int
	Appearance             Appearance
	Hits                   int
	TextPosition           int
	PhraseRelativePosition int
	PhrasePosition         int
}
