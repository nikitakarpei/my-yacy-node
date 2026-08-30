package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
)

const postingsBucket vault.Name = "rwi"

func registerPostings(
	v *vault.Vault,
) (*vault.Collection[postingIdentity, yacymodel.RWIPosting], error) {
	collection, err := v.RegisterCollection(
		postingsBucket,
		postingKeyCodec,
		postingValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register rwi posting collection: %w", err)
	}

	return collection, nil
}

var postingKeyLayout = vault.PairKey(hashcodec.Hash, hashcodec.URLHash)

var postingKeyCodec = postingKeyLayout.KeyCodecFor(
	func(posting postingIdentity) (yacymodel.Hash, yacymodel.URLHash) {
		return posting.word, posting.url
	},
	func(word yacymodel.Hash, url yacymodel.URLHash) postingIdentity {
		return postingIdentity{word: word, url: url}
	},
)

func everyPostingOf(word yacymodel.Hash) vault.KeyRange {
	return postingKeyLayout.KeysWithFirst(word)
}

type postingValueCodec struct{}

func (postingValueCodec) Encode(posting yacymodel.RWIPosting) ([]byte, error) {
	var w postingWriter
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
	r := newPostingReader(data)
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

func appearanceFields(a *yacymodel.Appearance) []*bool {
	return []*bool{
		&a.IndexOf,
		&a.HasLocation,
		&a.HasImage,
		&a.HasAudio,
		&a.HasVideo,
		&a.HasApp,
		&a.AppearsInDescription,
		&a.AppearsInTitle,
		&a.AppearsInCreator,
		&a.AppearsInSubject,
		&a.AppearsInIdentifier,
		&a.Emphasized,
	}
}

func packAppearance(a yacymodel.Appearance) uint16 {
	var bits uint16
	for position, field := range appearanceFields(&a) {
		if *field {
			bits |= 1 << uint(position)
		}
	}
	return bits
}

func unpackAppearance(bits uint16) yacymodel.Appearance {
	var a yacymodel.Appearance
	for position, field := range appearanceFields(&a) {
		*field = bits&(1<<uint(position)) != 0
	}
	return a
}
