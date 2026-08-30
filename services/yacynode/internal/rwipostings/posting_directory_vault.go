package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const postingsBucket vault.Name = "rwi"

func registerPostings(
	v *vault.Vault,
) (*vault.Collection[postingIdentity, yacymodel.RWIPosting], error) {
	collection, err := v.RegisterCollection(
		postingsBucket,
		postingKeyLayout,
		postingValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register rwi posting collection: %w", err)
	}

	return collection, nil
}

var postingKeyParts = vault.PairKey(hashkeypart.Hash, hashkeypart.URLHash)

var postingKeyLayout = postingKeyParts.KeyLayoutFor(
	func(posting postingIdentity) (yacymodel.Hash, yacymodel.URLHash) {
		return posting.word, posting.url
	},
	func(word yacymodel.Hash, url yacymodel.URLHash) postingIdentity {
		return postingIdentity{word: word, url: url}
	},
)

func everyPostingOf(word yacymodel.Hash) vault.KeyRange {
	return postingKeyParts.KeysWithFirst(word)
}

type postingValueCodec struct{}

func (postingValueCodec) Encode(posting yacymodel.RWIPosting) ([]byte, error) {
	var stored storedfields.Writer
	stored.Varint(int64(posting.LastModified))
	stored.Count(posting.TitleWords)
	stored.Count(posting.TextWords)
	stored.Count(posting.Phrases)
	stored.Byte(byte(posting.DocumentType))
	writeLanguage(&stored, posting.Language)
	stored.Count(posting.LocalLinks)
	stored.Count(posting.ExternalLinks)
	stored.Count(posting.URLLength)
	stored.Count(posting.URLComponents)
	stored.Uint16(packAppearance(posting.Appearance))
	stored.Count(posting.Hits)
	stored.Count(posting.TextPosition)
	stored.Count(posting.PhraseRelativePosition)
	stored.Count(posting.PhrasePosition)

	return stored.Record(), nil
}

func (postingValueCodec) Decode(data []byte) (yacymodel.RWIPosting, error) {
	if len(data) == 0 {
		return yacymodel.RWIPosting{}, fmt.Errorf(
			"%w: empty posting value",
			yacymodel.ErrBadRWIPosting,
		)
	}
	stored := storedfields.ReaderOf(data, yacymodel.ErrBadRWIPosting)
	posting := yacymodel.RWIPosting{
		LastModified:           yacymodel.MicroDate(stored.Varint("last modified")),
		TitleWords:             stored.Count("title words"),
		TextWords:              stored.Count("text words"),
		Phrases:                stored.Count("phrases"),
		DocumentType:           yacymodel.DocumentType(stored.Byte("document type")),
		Language:               languageFrom(stored),
		LocalLinks:             stored.Count("local links"),
		ExternalLinks:          stored.Count("external links"),
		URLLength:              stored.Count("url length"),
		URLComponents:          stored.Count("url components"),
		Appearance:             unpackAppearance(stored.Uint16("appearance")),
		Hits:                   stored.Count("hits"),
		TextPosition:           stored.Count("text position"),
		PhraseRelativePosition: stored.Count("phrase relative position"),
		PhrasePosition:         stored.Count("phrase position"),
	}
	if err := stored.Err(); err != nil {
		return yacymodel.RWIPosting{}, err
	}

	return posting, nil
}

func writeLanguage(
	stored *storedfields.Writer,
	language yacymodel.Optional[yacymodel.Language],
) {
	code, spoken := language.Get()
	if !spoken {
		stored.Text("")

		return
	}
	stored.Text(code.String())
}

func languageFrom(stored *storedfields.Reader) yacymodel.Optional[yacymodel.Language] {
	raw := stored.Text("language")
	if raw == "" {
		return yacymodel.None[yacymodel.Language]()
	}
	language, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		stored.Reject("language", err)

		return yacymodel.None[yacymodel.Language]()
	}

	return yacymodel.Some(language)
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
