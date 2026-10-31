package rwipostings

import (
	"context"
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const (
	postingsBucket    vault.Name = "rwi"
	urlWordsBucket    vault.Name = "rwi_url_words"
	urlPostingsBucket vault.Name = "rwi_url_postings"
)

const (
	urlWordsVersion    = "1"
	urlPostingsVersion = "1"
)

type urlPostingsIndex = vault.ValueIndex[yacymodel.RWIPosting, yacymodel.RWIPosting]

func registerPostings(
	ctx context.Context,
	v *vault.Vault,
	rankedWords []yacymodel.Hash,
	urlWordsObserver vault.MemberSetObserver,
	projectionsOfConsumers []vault.Projection[yacymodel.RWIPosting],
) (
	*vault.IndexedCollection[postingIdentity, yacymodel.RWIPosting],
	*vault.MemberSetIndex[yacymodel.Hash, yacymodel.URLHash, postingIdentity],
	*urlPostingsIndex,
	error,
) {
	wordRanking, err := vault.FixedDictionaryFrom(hashkeypart.Hash, rankedWords)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("rank words: %w", err)
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
		return nil, nil, nil, fmt.Errorf("register url words: %w", err)
	}
	urlPostings, err := v.RegisterValueIndex(
		urlPostingsBucket,
		urlPostingsVersion,
		urlPostingKeyLayout,
		func(posting yacymodel.RWIPosting) yacymodel.RWIPosting { return posting },
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("register url postings: %w", err)
	}
	postingProjections := append(slices.Clone(projectionsOfConsumers), urlPostings)
	postings, err := v.RegisterIndexedCollection(
		ctx,
		postingsBucket,
		urlWordsIndex,
		postingValueCodec{},
		vault.ProjectedAs(postingWithIdentity, postingProjections...),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("register rwi posting collection: %w", err)
	}

	return postings, urlWordsIndex, urlPostings, nil
}

var postingKeyParts = vault.PairKey(hashkeypart.Hash, hashkeypart.URLHash)

var postingFieldsKeyPart = vault.BytesKeyPartFrom(appendPostingFields, postingFieldsFrom)

var urlPostingKeyParts = vault.TripleKey(
	hashkeypart.URLHash,
	hashkeypart.Hash,
	postingFieldsKeyPart,
)

var urlPostingKeyLayout = urlPostingKeyParts.KeyLayoutFor(
	func(posting yacymodel.RWIPosting) (yacymodel.URLHash, yacymodel.Hash, yacymodel.RWIPosting) {
		return posting.URLHash, posting.WordHash, posting
	},
	func(
		url yacymodel.URLHash,
		word yacymodel.Hash,
		fields yacymodel.RWIPosting,
	) yacymodel.RWIPosting {
		fields.URLHash, fields.WordHash = url, word

		return fields
	},
)

func appendPostingFields(posting yacymodel.RWIPosting, encoded []byte) []byte {
	return append(encoded, postingFieldsOf(posting)...)
}

func postingFieldsFrom(stored vault.StoredBytes) (yacymodel.RWIPosting, error) {
	return postingValueCodec{}.Decode(stored.AppendTo(nil))
}

func postingsOfWordBetween(
	word yacymodel.Hash,
	lowestURL, highestURL yacymodel.URLHash,
) vault.KeyRange {
	return postingKeyParts.KeysWithFirstBetweenSeconds(word, lowestURL, highestURL)
}

type postingValueCodec struct{}

func (postingValueCodec) Encode(posting yacymodel.RWIPosting) ([]byte, error) {
	return postingFieldsOf(posting), nil
}

func postingFieldsOf(posting yacymodel.RWIPosting) []byte {
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

	return stored.Record()
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
