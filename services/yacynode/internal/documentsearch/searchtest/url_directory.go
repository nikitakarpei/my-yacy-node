package searchtest

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type URLDirectory struct {
	Documents map[yacymodel.URLHash]yacymodel.URLMetadata
}

func (d URLDirectory) MetadataByHash(
	_ *vault.Txn,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, error) {
	out := make([]yacymodel.URLMetadata, 0, len(hashes))
	for _, hash := range hashes {
		if stored, ok := d.Documents[hash]; ok {
			out = append(out, stored)
		}
	}

	return out, nil
}

func (d URLDirectory) Count(*vault.Txn) (int, error) {
	return len(d.Documents), nil
}

type FailingURLDirectory struct {
	Err error
}

func (d FailingURLDirectory) MetadataByHash(
	*vault.Txn,
	[]yacymodel.URLHash,
) ([]yacymodel.URLMetadata, error) {
	return nil, d.Err
}

func (d FailingURLDirectory) Count(*vault.Txn) (int, error) {
	return 0, d.Err
}
