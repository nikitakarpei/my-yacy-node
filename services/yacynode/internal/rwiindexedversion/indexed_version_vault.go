package rwiindexedversion

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

const indexedVersionBucket vault.Name = "rwi_indexed_versions"

var errBadIndexedVersion = errors.New("bad indexed version")

func registerIndexedVersions(
	v *vault.Vault,
) (*vault.Collection[yacymodel.URLHash, IndexedVersion], error) {
	indexed, err := v.RegisterCollection(
		indexedVersionBucket,
		vault.SingleKey(hashkeypart.URLHash).KeyLayout(),
		indexedVersionCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register indexed versions: %w", err)
	}

	return indexed, nil
}

type indexedVersionCodec struct{}

func (indexedVersionCodec) Encode(indexed IndexedVersion) ([]byte, error) {
	var stored storedfields.Writer
	fingerprint := indexed.Fingerprint
	stored.Varint(int64(fingerprint.Modified))
	stored.Count(fingerprint.LocalLinks)
	stored.Count(fingerprint.ExternalLinks)
	stored.Count(fingerprint.TitleWords)
	stored.Text(fingerprint.Language.String())
	stored.Byte(byte(fingerprint.DocumentType))
	rwiorigin.Write(&stored, indexed.Origin)

	return stored.Record(), nil
}

func (indexedVersionCodec) Decode(raw []byte) (IndexedVersion, error) {
	stored := storedfields.ReaderOf(raw, errBadIndexedVersion)
	fingerprint := pagefingerprint.Fingerprint{
		Modified:      yacymodel.MicroDate(stored.Varint("modified")),
		LocalLinks:    stored.Count("local links"),
		ExternalLinks: stored.Count("external links"),
		TitleWords:    stored.Count("title words"),
		Language:      languageFrom(stored),
		DocumentType:  yacymodel.DocumentType(stored.Byte("document type")),
	}
	origin := rwiorigin.From(stored)
	if err := stored.Err(); err != nil {
		return IndexedVersion{}, err
	}

	return IndexedVersion{Fingerprint: fingerprint, Origin: origin}, nil
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
