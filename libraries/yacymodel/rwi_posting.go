package yacymodel

import "errors"

// ErrBadRWIPosting reports a posting that cannot be read back into the domain
// type, whether it came off the DHT wire or off local disk.
var ErrBadRWIPosting = errors.New("bad rwi posting")

type RWIPosting struct {
	WordHash               Hash
	URLHash                URLHash
	LastModified           MicroDate
	Language               Language
	DocumentType           DocumentType
	TitleWords             uint8
	TextWords              uint16
	Phrases                uint16
	LocalLinks             uint8
	ExternalLinks          uint8
	URLLength              uint8
	URLComponents          uint8
	Appearance             Appearance
	TextPosition           uint16
	Hits                   uint8
	PhraseRelativePosition uint8
	PhrasePosition         uint8
}
