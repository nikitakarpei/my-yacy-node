package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const storedPostingFormat byte = 0x02

type postingCodec struct{}

func (postingCodec) Encode(posting yacymodel.RWIPosting) ([]byte, error) {
	var w postingWriter
	w.uint8(storedPostingFormat)
	w.fixed([]byte(posting.URLHash.String()))
	w.varint(int64(posting.LastModified))
	w.uint8(posting.TitleWords)
	w.uint16(posting.TextWords)
	w.uint16(posting.Phrases)
	w.uint8(byte(posting.DocumentType))
	w.lengthPrefixed([]byte(posting.Language))
	w.uint8(posting.LocalLinks)
	w.uint8(posting.ExternalLinks)
	w.uint8(posting.URLLength)
	w.uint8(posting.URLComponents)
	w.uint16(packAppearance(posting.Appearance))
	w.uint8(posting.Hits)
	w.uint16(posting.TextPosition)
	w.uint8(posting.PhraseRelativePosition)
	w.uint8(posting.PhrasePosition)

	return w.bytes(), nil
}

func (postingCodec) Decode(data []byte) (yacymodel.RWIPosting, error) {
	if len(data) == 0 {
		return yacymodel.RWIPosting{}, fmt.Errorf(
			"%w: empty posting value",
			yacymodel.ErrBadRWIPosting,
		)
	}
	if data[0] != storedPostingFormat {
		return yacymodel.RWIPosting{}, fmt.Errorf(
			"%w: unknown stored posting format 0x%02x, want 0x%02x",
			yacymodel.ErrBadRWIPosting,
			data[0],
			storedPostingFormat,
		)
	}

	r := newPostingReader(data[1:])
	rawURLHash := r.fixed("url hash", yacymodel.HashLength)
	posting := yacymodel.RWIPosting{
		LastModified:           yacymodel.MicroDate(r.varint("last modified")),
		TitleWords:             r.uint8("title words"),
		TextWords:              r.uint16("text words"),
		Phrases:                r.uint16("phrases"),
		DocumentType:           yacymodel.DocumentType(r.uint8("document type")),
		Language:               yacymodel.Language(r.lengthPrefixed("language")),
		LocalLinks:             r.uint8("local links"),
		ExternalLinks:          r.uint8("external links"),
		URLLength:              r.uint8("url length"),
		URLComponents:          r.uint8("url components"),
		Appearance:             unpackAppearance(r.uint16("appearance")),
		Hits:                   r.uint8("hits"),
		TextPosition:           r.uint16("text position"),
		PhraseRelativePosition: r.uint8("phrase relative position"),
		PhrasePosition:         r.uint8("phrase position"),
	}
	if r.err != nil {
		return yacymodel.RWIPosting{}, r.err
	}

	urlHash, err := yacymodel.ParseURLHash(string(rawURLHash))
	if err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("%w: %w", yacymodel.ErrBadRWIPosting, err)
	}
	posting.URLHash = urlHash

	return posting, nil
}
