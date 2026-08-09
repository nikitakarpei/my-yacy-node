package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const storedPostingFormat byte = 0x02

type postingValueCodec struct{}

func (postingValueCodec) Encode(posting yacymodel.RWIPosting) ([]byte, error) {
	var w postingWriter
	w.uint8(storedPostingFormat)
	w.fixed([]byte(posting.URLHash.String()))
	w.varint(int64(posting.LastModified))
	w.count(posting.TitleWords)
	w.count(posting.TextWords)
	w.count(posting.Phrases)
	w.uint8(byte(posting.DocumentType))
	var languageCode []byte
	if language, ok := posting.Language.Get(); ok {
		languageCode = []byte(language.String())
	}
	w.lengthPrefixed(languageCode)
	w.count(posting.LocalLinks)
	w.count(posting.ExternalLinks)
	w.count(posting.URLLength)
	w.count(posting.URLComponents)
	w.uint16(packAppearance(posting.Appearance))
	w.count(posting.Hits)
	w.count(posting.TextPosition)
	w.count(posting.PhraseRelativePosition)
	w.count(posting.PhrasePosition)

	return w.bytes(), nil
}

func (postingValueCodec) Decode(data []byte) (yacymodel.RWIPosting, error) {
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
		TitleWords:             r.count("title words"),
		TextWords:              r.count("text words"),
		Phrases:                r.count("phrases"),
		DocumentType:           yacymodel.DocumentType(r.uint8("document type")),
		Language:               r.language("language"),
		LocalLinks:             r.count("local links"),
		ExternalLinks:          r.count("external links"),
		URLLength:              r.count("url length"),
		URLComponents:          r.count("url components"),
		Appearance:             unpackAppearance(r.uint16("appearance")),
		Hits:                   r.count("hits"),
		TextPosition:           r.count("text position"),
		PhraseRelativePosition: r.count("phrase relative position"),
		PhrasePosition:         r.count("phrase position"),
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
