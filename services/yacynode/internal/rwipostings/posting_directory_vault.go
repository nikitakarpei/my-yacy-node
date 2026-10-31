package rwipostings

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const (
	postingsBucket vault.Name = "rwi"
	urlWordsBucket vault.Name = "rwi_url_words"
)

const urlWordsVersion = "1"

func registerPostings(
	ctx context.Context,
	v *vault.Vault,
	rankedWords []yacymodel.Hash,
	urlWordsObserver vault.MemberSetObserver,
	projectionsOfConsumers []vault.Projection[yacymodel.RWIPosting],
) (
	*vault.IndexedCollection[postingIdentity, yacymodel.RWIPosting],
	*vault.MemberSetIndex[yacymodel.Hash, yacymodel.URLHash, postingIdentity],
	error,
) {
	wordRanking, err := vault.FixedDictionaryFrom(hashkeypart.Hash, rankedWords)
	if err != nil {
		return nil, nil, fmt.Errorf("rank words: %w", err)
	}
	urlWordsIndex, err := v.RegisterMemberSetIndex(
		urlWordsBucket,
		urlWordsVersion,
		postingKeyParts,
		func(posting postingIdentity) (yacymodel.Hash, yacymodel.URLHash) {
			return posting.word, posting.url
		},
		func(word yacymodel.Hash, url yacymodel.URLHash) postingIdentity {
			return postingIdentity{word: word, url: url}
		},
		vault.MemberEncodingFrom(wordRanking),
		urlWordsObserver,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register url words: %w", err)
	}
	postings, err := v.RegisterIndexedCollection(
		ctx,
		postingsBucket,
		urlWordsIndex,
		postingValueCodec{},
		vault.ProjectedAs(postingWithIdentity, projectionsOfConsumers...),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register rwi posting collection: %w", err)
	}

	return postings, urlWordsIndex, nil
}

var postingKeyParts = vault.PairKey(hashkeypart.Hash, hashkeypart.URLHash)

func postingsOfWordBetween(
	word yacymodel.Hash,
	lowestURL, highestURL yacymodel.URLHash,
) vault.KeyRange {
	return postingKeyParts.KeysWithFirstBetweenSeconds(word, lowestURL, highestURL)
}

type postingValueCodec struct{}

func (postingValueCodec) Encode(posting yacymodel.RWIPosting) ([]byte, error) {
	var stored storedfields.Writer
	stored.Varint(int64(posting.LastModified))
	stored.Count(int(posting.TitleWords))
	stored.Count(int(posting.TextWords))
	stored.Count(int(posting.Phrases))
	stored.Byte(byte(posting.DocumentType))
	stored.Text(posting.Language.String())
	stored.Count(int(posting.LocalLinks))
	stored.Count(int(posting.ExternalLinks))
	stored.Count(int(posting.URLLength))
	stored.Count(int(posting.URLComponents))
	stored.Uint16(packAppearance(posting.Appearance))
	stored.Count(int(posting.Hits))
	stored.Count(int(posting.TextPosition))
	stored.Count(int(posting.PhraseRelativePosition))
	stored.Count(int(posting.PhrasePosition))

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
		LastModified:  yacymodel.MicroDateFromDays(stored.Varint("last modified")).OrElse(0),
		TitleWords:    yacymodel.WrappedCardinalFrom(stored.Count("title words")),
		TextWords:     yacymodel.SaturatedCardinalFrom[uint16](stored.Count("text words")),
		Phrases:       yacymodel.SaturatedCardinalFrom[uint16](stored.Count("phrases")),
		DocumentType:  yacymodel.DocumentType(stored.Byte("document type")),
		Language:      languageFrom(stored),
		LocalLinks:    yacymodel.WrappedCardinalFrom(stored.Count("local links")),
		ExternalLinks: yacymodel.WrappedCardinalFrom(stored.Count("external links")),
		URLLength:     yacymodel.WrappedCardinalFrom(stored.Count("url length")),
		URLComponents: yacymodel.WrappedCardinalFrom(stored.Count("url components")),
		Appearance:    unpackAppearance(stored.Uint16("appearance")),
		Hits:          yacymodel.WrappedCardinalFrom(stored.Count("hits")),
		TextPosition:  yacymodel.SaturatedCardinalFrom[uint16](stored.Count("text position")),
		PhraseRelativePosition: yacymodel.WrappedCardinalFrom(
			stored.Count("phrase relative position"),
		),
		PhrasePosition: yacymodel.WrappedCardinalFrom(stored.Count("phrase position")),
	}
	if err := stored.Err(); err != nil {
		return yacymodel.RWIPosting{}, err
	}

	return posting, nil
}

func languageFrom(stored *storedfields.Reader) yacymodel.Language {
	var code [yacymodel.LanguageCodeLength]byte
	language, err := yacymodel.ParseLanguageBytes(stored.AppendBytes(code[:0], "language"))
	if err != nil {
		stored.Reject("language", err)

		return yacymodel.Language{}
	}

	return language
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
