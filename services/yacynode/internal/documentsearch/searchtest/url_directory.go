package searchtest

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type URLDirectory struct {
	Documents map[yacymodel.URLHash]yacymodel.URLMetadata
}

func (d URLDirectory) MetadataPerHash(
	_ *vault.Txn,
	hashes []yacymodel.URLHash,
) (map[yacymodel.URLHash]yacymodel.URLMetadata, error) {
	found := make(map[yacymodel.URLHash]yacymodel.URLMetadata, len(hashes))
	for _, hash := range hashes {
		if stored, ok := d.Documents[hash]; ok {
			found[hash] = stored
		}
	}

	return found, nil
}

func (d URLDirectory) Count(*vault.Txn) (int, error) {
	return len(d.Documents), nil
}

type FailingURLDirectory struct {
	Err error
}

func (d FailingURLDirectory) MetadataPerHash(
	*vault.Txn,
	[]yacymodel.URLHash,
) (map[yacymodel.URLHash]yacymodel.URLMetadata, error) {
	return nil, d.Err
}

func (d FailingURLDirectory) Count(*vault.Txn) (int, error) {
	return 0, d.Err
}
