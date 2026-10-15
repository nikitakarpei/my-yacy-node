package urlpageversion

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

const indexedVersionBucket vault.Name = "url_page_versions"

var errBadIndexedVersion = errors.New("bad indexed page version")

func registerIndexedVersions(
	v *vault.Vault,
) (*vault.Collection[yacymodel.URLHash, indexedVersion], error) {
	indexed, err := v.RegisterCollection(
		indexedVersionBucket,
		vault.SingleKey(hashkeypart.URLHash).KeyLayout(),
		indexedVersionCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register indexed page versions: %w", err)
	}

	return indexed, nil
}

type indexedVersionCodec struct{}

func (indexedVersionCodec) Encode(indexed indexedVersion) ([]byte, error) {
	var stored storedfields.Writer
	stored.Varint(int64(indexed.Version))
	rwiorigin.Write(&stored, indexed.Origin)

	return stored.Record(), nil
}

func (indexedVersionCodec) Decode(raw []byte) (indexedVersion, error) {
	stored := storedfields.ReaderOf(raw, errBadIndexedVersion)
	version := yacymodel.MicroDate(stored.Varint("version"))
	origin := rwiorigin.From(stored)
	if err := stored.Err(); err != nil {
		return indexedVersion{}, err
	}

	return indexedVersion{Version: version, Origin: origin}, nil
}
