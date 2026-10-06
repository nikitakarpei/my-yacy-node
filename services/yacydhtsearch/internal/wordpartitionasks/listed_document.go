package wordpartitionasks

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type ListedDocument struct {
	Hash     yacymodel.URLHash
	metadata *yacymodel.URLMetadata
	posting  *yacymodel.RWIPosting
}

func ListedDocumentFrom(
	metadata yacymodel.URLMetadata,
	posting yacymodel.Optional[yacymodel.RWIPosting],
) ListedDocument {
	listedDocument := ListedDocument{Hash: metadata.Hash, metadata: &metadata}
	if sentPosting, sent := posting.Get(); sent {
		listedDocument.posting = &sentPosting
	}

	return listedDocument
}

func (listedDocument ListedDocument) Metadata() yacymodel.Optional[yacymodel.URLMetadata] {
	if listedDocument.metadata == nil {
		return yacymodel.None[yacymodel.URLMetadata]()
	}

	return yacymodel.Some(*listedDocument.metadata)
}

func (listedDocument ListedDocument) Posting() yacymodel.Optional[yacymodel.RWIPosting] {
	if listedDocument.posting == nil {
		return yacymodel.None[yacymodel.RWIPosting]()
	}

	return yacymodel.Some(*listedDocument.posting)
}
