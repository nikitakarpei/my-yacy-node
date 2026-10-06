package queryfindings

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type DocumentsThePeersSent struct {
	documentsInFoundOrder       []yacymodel.URLHash
	metadataReplicasPerDocument map[yacymodel.URLHash][]MetadataReplica
	postingReplicasPerDocument  map[yacymodel.URLHash][]PostingReplica
}

func EmptyDocumentsThePeersSent() *DocumentsThePeersSent {
	return &DocumentsThePeersSent{
		metadataReplicasPerDocument: map[yacymodel.URLHash][]MetadataReplica{},
		postingReplicasPerDocument:  map[yacymodel.URLHash][]PostingReplica{},
	}
}

func (documents *DocumentsThePeersSent) KeepMetadataReplica(replica MetadataReplica) {
	document := replica.Metadata.Hash
	if _, alreadyFound := documents.metadataReplicasPerDocument[document]; !alreadyFound {
		documents.documentsInFoundOrder = append(documents.documentsInFoundOrder, document)
	}
	documents.metadataReplicasPerDocument[document] = append(
		documents.metadataReplicasPerDocument[document], replica,
	)
}

func (documents *DocumentsThePeersSent) KeepPostingReplica(replica PostingReplica) {
	document := replica.Posting.URLHash
	documents.postingReplicasPerDocument[document] = append(
		documents.postingReplicasPerDocument[document], replica,
	)
}

func (documents *DocumentsThePeersSent) FoundDocuments() []FoundDocument {
	if len(documents.documentsInFoundOrder) == 0 {
		return nil
	}

	foundDocuments := make([]FoundDocument, 0, len(documents.documentsInFoundOrder))
	for _, document := range documents.documentsInFoundOrder {
		foundDocuments = append(foundDocuments, FoundDocumentOf(
			document,
			documents.metadataReplicasPerDocument[document],
			documents.postingReplicasPerDocument[document],
		))
	}

	return foundDocuments
}
