package queryanswers

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type MetadataPerDocument map[yacymodel.URLHash]yacymodel.URLMetadata

func (m MetadataPerDocument) Keep(metadata yacymodel.URLMetadata) {
	if _, alreadyKept := m[metadata.Hash]; alreadyKept {
		return
	}
	m[metadata.Hash] = metadata
}

func (m MetadataPerDocument) withoutDocuments(
	documents map[yacymodel.URLHash]struct{},
) MetadataPerDocument {
	metadataPerDocument := make(MetadataPerDocument, len(m))
	for document, metadata := range m {
		if _, left := documents[document]; left {
			continue
		}
		metadataPerDocument[document] = metadata
	}

	return metadataPerDocument
}
