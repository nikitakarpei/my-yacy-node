package yacymodel

import "errors"

// ErrBadRWIPosting reports a posting that cannot be read back into the domain
// type, whether it came off the DHT wire or off local disk.
var ErrBadRWIPosting = errors.New("bad rwi posting")

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
